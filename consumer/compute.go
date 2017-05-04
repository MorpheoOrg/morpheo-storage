package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/MorpheoOrg/morpheo-compute/common"
)

// Worker describes a worker (where it stores its data, which container runtime it uses...).
// Most importantly, it carefully implements all the steps of our learning/testing/prediction
// workflow.
//
// For an in-detail understanding of what these different steps do and how, check out Camille's
// awesome example: https://github.com/MorpheoOrg/hypnogram-wf
// The doc also gets there in detail: https://morpheoorg.github.io/morpheo/modules/learning.html
type Worker struct {
	dataFolder           string
	trainFolder          string
	testFolder           string
	untargetedTestFolder string
	predFolder           string
	problemImagePrefix   string
	modelImagePrefix     string

	containerRuntime common.ContainerRuntime
	storage          common.StorageBackend
	// TODO: add the orchestrator here and talk with him
}

// NewWorker creates a Worker instance
func NewWorker(dataFolder, trainFolder, testFolder, untargetedTestFolder, predFolder, problemImagePrefix, modelImagePrefix string, containerRuntime common.ContainerRuntime, storage common.StorageBackend) *Worker {
	return &Worker{
		dataFolder:           dataFolder,
		trainFolder:          trainFolder,
		testFolder:           testFolder,
		predFolder:           predFolder,
		untargetedTestFolder: untargetedTestFolder,
		problemImagePrefix:   problemImagePrefix,
		modelImagePrefix:     modelImagePrefix,

		containerRuntime: containerRuntime,
		storage:          storage,
	}
}

// HandleLearn handles learning tasks
func (w *Worker) HandleLearn(message []byte) (err error) {
	var task common.LearnUplet
	err = json.NewDecoder(bytes.NewReader(message)).Decode(&task)
	if err != nil {
		return fmt.Errorf("Error un-marshaling learn-uplet: %s -- Body: %s", err, message)
	}

	if err = task.Check(); err != nil {
		return fmt.Errorf("Error in train task: %s -- Body: %s", err, message)
	}

	problemWorkflow, err := w.storage.GetProblemWorkflow(task.Problem)
	defer problemWorkflow.Close()
	if err != nil {
		return fmt.Errorf("Error pulling problem workflow %s from storage: %s", task.Problem, err)
	}

	problemImageName := fmt.Sprintf("%s-%s", w.problemImagePrefix, task.Problem)
	err = w.ProblemWorkflowImageLoad(problemImageName, problemWorkflow)
	if err != nil {
		return fmt.Errorf("Error loading problem workflow image %s in Docker daemon: %s", task.Problem, err)
	}

	model, err := w.storage.GetModel(task.ModelStart)
	defer model.Close()
	if err != nil {
		return fmt.Errorf("Error pulling model %s from storage: %s", task.ModelStart, err)
	}

	modelImageName := fmt.Sprintf("%s-%s", w.modelImagePrefix, task.ModelStart)
	err = w.ModelImageLoad(modelImageName, model)
	if err != nil {
		return fmt.Errorf("Error loading model image %s in Docker daemon: %s", modelImageName, err)
	}

	// Setup directory structure
	taskDataFolder := fmt.Sprintf("%s/%s", w.dataFolder, task.ModelStart)
	trainFolder := fmt.Sprintf("%s/%s/%s", taskDataFolder, w.trainFolder)
	testFolder := fmt.Sprintf("%s/%s/%s", taskDataFolder, w.testFolder)
	untargetedTestFolder := fmt.Sprintf("%s/%s/%s", taskDataFolder, w.untargetedTestFolder)
	// TODO: set that up

	// Pulling train dataset
	for _, dataID := range task.TrainData {
		data, err := w.storage.GetData(dataID)
		if err != nil {
			return fmt.Errorf("Error pulling train dataset %s from storage: %s", dataID, err)
		}
		path := fmt.Sprintf("%s/%s", trainFolder, dataID)
		dataFile, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("Error creating file %s: %s", path, err)
		}
		n, err := io.Copy(dataFile, data)
		if err != nil {
			return fmt.Errorf("Error copying train data file %s (%d bytes written): %s", path, n, err)
		}
		dataFile.Close()
		data.Close()
	}

	// And the test data
	for _, dataID := range task.TestData {
		data, err := w.storage.GetData(dataID)
		if err != nil {
			return fmt.Errorf("Error pulling test dataset %s from storage: %s", dataID, err)
		}
		path := fmt.Sprintf("%s/%s", testFolder, dataID)
		dataFile, err := os.Create(path)
		n, err := io.Copy(dataFile, data)
		if err != nil {
			return fmt.Errorf("Error copying test data file %s (%d bytes written): %s", path, n, err)
		}
		dataFile.Close()
		data.Close()
	}

	// Let's remove targets from the test data
	err = w.UntargetTestingVolume(problemImageName, testFolder, untargetedTestFolder)
	if err != nil {
		return fmt.Errorf("Error preparing problem %s for %s: %s", task.Problem, task.ModelStart, err)
	}

	// Let's pass the task to our execution backend, now that everything should be in place
	containerID, err := w.Train(modelImageName, trainFolder)
	if err != nil {
		return fmt.Errorf("Error in train task: %s -- Body: %s", err, message)
	}
	err = w.Predict(modelImageName, untargetedTestFolder)
	if err != nil {
		return fmt.Errorf("Error in test task: %s -- Body: %s", err, message)
	}

	// Let's move test predictions to the test folder with targets
	os.Rename(
		fmt.Sprintf("%s/%s", untargetedTestFolder, w.predFolder),
		fmt.Sprintf("%s/%s", testFolder, w.predFolder),
	)

	// Let's compute the performance !
	newModelImageName := fmt.Sprintf("%s-%s", w.modelImagePrefix, task.ModelEnd)
	err = w.ComputePerf(problemImageName, trainFolder, testFolder)
	if err != nil {
		// FIXME: do not return here
		return fmt.Errorf("Error computing perf for problem %s and model (new) %s: %s", task.Problem, task.ModelEnd, err)
	}

	// Let's create a new model and post it to storage
	snapshot, err := w.containerRuntime.SnapshotContainer(containerID, newModelImageName)
	if err != nil {
		return fmt.Errorf("Error snapshotting container %s to image %s: %s", containerID, newModelImageName, err)
	}

	err = w.storage.PostModel(task.ModelEnd, snapshot)
	if err != nil {
		return fmt.Errorf("Error streaming new model %s to storage: %s", task.ModelEnd, err)
	}

	// TODO: parse the perf file and notify the orchestrator about it

	log.Printf("[INFO] Train finished with success, cleaning up...")

	err = os.RemoveAll(taskDataFolder)
	if err != nil {
		return fmt.Errorf("Error removing data folder %s: %s", taskDataFolder, err)
	}

	err = w.containerRuntime.ImageUnload(modelImageName)
	if err != nil {
		return fmt.Errorf("Error unloading image %s: %s", modelImageName, err)
	}

	err = w.containerRuntime.ImageUnload(problemImageName)
	if err != nil {
		return fmt.Errorf("Error unloading image %s: %s", problemImageName, err)
	}

	return
}

// HandlePred handles our prediction tasks
// func (w *Worker) HandlePred(message []byte) (err error) {
// 	var task common.Preduplet
// 	err = json.NewDecoder(bytes.NewReader(message)).Decode(&task)
// 	if err != nil {
// 		return fmt.Errorf("Error un-marshaling pred-uplet: %s -- Body: %s", err, message)
// 	}
//
// 	// Let's pass the prediction task to our execution backend
// 	prediction, err := w.executionBackend.Predict(task.Model, task.Data)
// 	if err != nil {
// 		return fmt.Errorf("Error in prediction task: %s -- Body: %s", err, message)
// 	}
//
// 	// TODO: send the prediction to the viewer, asynchronously
// 	log.Printf("Predicition completed with success. Predicition %s", prediction)
//
// 	return
// }

// ProblemWorkflowImageLoad loads the docker image corresponding to a problem workflow in the Docker
// daemon that will then run this problem workflow
func (w *Worker) ProblemWorkflowImageLoad(problemImage string, imageReader io.Reader) error {
	return w.containerRuntime.ImageLoad(problemImage, imageReader)
}

// ModelImageLoad loads the Docker image corresponding to a given model
func (w *Worker) ModelImageLoad(modelImage string, imageReader io.Reader) error {
	return w.containerRuntime.ImageLoad(modelImage, imageReader)
}

// UntargetTestingVolume copies data from /<host-data-volume>/<model>/data to
// /<host-data-volume>/<model>/train and removes targets from test files... using the problem
// workflow container.
func (w *Worker) UntargetTestingVolume(problemImage, testFolder, untargetedTestFolder string) error {
	_, err := w.containerRuntime.RunImageInUntrustedContainer(
		problemImage,
		[]string{"-T", "detarget", "-o", "/true_data", "-p", "/pred_data"},
		map[string]string{
			testFolder:           "/true_data",
			untargetedTestFolder: "/pred_data",
		}, true)
	return err
}

// Train launches the submission container's train routines
func (w *Worker) Train(modelImage, trainFolder string) (containerID string, err error) {
	return w.containerRuntime.RunImageInUntrustedContainer(
		modelImage,
		[]string{"-V", "/data", "-T", "train"},
		map[string]string{
			trainFolder: "/data/train",
		}, false)
}

// Predict launches the submission container's predict routines
func (w *Worker) Predict(modelImage, testFolder string) error {
	_, err := w.containerRuntime.RunImageInUntrustedContainer(
		modelImage,
		[]string{"-V", "/data", "-T", "predict"},
		map[string]string{
			testFolder: "/true_data",
		}, true)
	return err
}

// ComputePerf analyses the prediction folders and computes a score for the model
func (w *Worker) ComputePerf(problemImage, trainFolder, testFolder string) error {
	_, err := w.containerRuntime.RunImageInUntrustedContainer(
		problemImage,
		[]string{"-T", "perf", "-o", "/true_data", "-p", "/pred_data"},
		map[string]string{
			trainFolder: "/true_data",
			testFolder:  "/pred_data",
		}, true)
	return err
}

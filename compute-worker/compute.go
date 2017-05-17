package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/MorpheoOrg/go-morpheo/client"
	"github.com/MorpheoOrg/go-morpheo/common"
)

// Worker describes a worker (where it stores its data, which container runtime it uses...).
// Most importantly, it carefully implements all the steps of our learning/testing/prediction
// workflow.
//
// For an in-detail understanding of what these different steps do and how, check out Camille's
// awesome example: https://github.com/MorpheoOrg/hypnogram-wf
// The doc also gets there in detail: https://morpheoorg.github.io/morpheo/modules/learning.html
type Worker struct {
	// Worker configuration variables
	dataFolder           string
	trainFolder          string
	testFolder           string
	untargetedTestFolder string
	modelFolder          string
	predFolder           string
	problemImagePrefix   string
	modelImagePrefix     string

	// ContainerRuntime abstractions
	containerRuntime common.ContainerRuntime

	// Morpheo API clients
	storage      client.Storage
	orchestrator client.Orchestrator
}

// NewWorker creates a Worker instance
func NewWorker(dataFolder, trainFolder, testFolder, untargetedTestFolder, predFolder, modelFolder, problemImagePrefix, modelImagePrefix string, containerRuntime common.ContainerRuntime, storage client.Storage, orchestrator client.Orchestrator) *Worker {
	return &Worker{
		dataFolder:           dataFolder,
		trainFolder:          trainFolder,
		testFolder:           testFolder,
		predFolder:           predFolder,
		untargetedTestFolder: untargetedTestFolder,
		modelFolder:          modelFolder,
		problemImagePrefix:   problemImagePrefix,
		modelImagePrefix:     modelImagePrefix,

		containerRuntime: containerRuntime,
		storage:          storage,
		orchestrator:     orchestrator,
	}
}

// HandleLearn manages a learning task (orchestrator status updates, etc...)
func (w *Worker) HandleLearn(message []byte) (err error) {

	// Unmarshal the learn-uplet
	var task common.LearnUplet

	err = json.NewDecoder(bytes.NewReader(message)).Decode(&task)
	if err != nil {
		return fmt.Errorf("Error un-marshaling learn-uplet: %s -- Body: %s", err, message)
	}

	if err = task.Check(); err != nil {
		return fmt.Errorf("Error in train task: %s -- Body: %s", err, message)
	}

	// Update its status to pending on the orchestrator
	w.orchestrator.UpdateUpletStatus(common.TypeLearnUplet, common.TaskStatusPending, task.ID)

	err = w.LearnWorkflow(task)
	if err != nil {
		// TODO: handle fatal and non-fatal errors differently and set learnuplet status to failed only
		// if the error was fatal
		w.orchestrator.UpdateUpletStatus(common.TypeLearnUplet, common.TaskStatusFailed, task.ID)
		return fmt.Errorf("Error in LearnWorkflow: %s", err)
	}

	w.orchestrator.UpdateUpletStatus(common.TypeLearnUplet, common.TaskStatusDone, task.ID)
	return nil
}

// LearnWorkflow implements our learning workflow
func (w *Worker) LearnWorkflow(task common.LearnUplet) (err error) {
	problemWorkflow, err := w.storage.GetProblemWorkflowBlob(task.Problem)
	if err != nil {
		return fmt.Errorf("Error pulling problem workflow %s from storage: %s", task.Problem, err)
	}
	defer problemWorkflow.Close()

	problemImageName := fmt.Sprintf("%s-%s", w.problemImagePrefix, task.Problem)
	err = w.ProblemWorkflowImageLoad(problemImageName, problemWorkflow)
	if err != nil {
		return fmt.Errorf("Error loading problem workflow image %s in Docker daemon: %s", task.Problem, err)
	}
	defer w.containerRuntime.ImageUnload(problemImageName)

	model, err := w.storage.GetAlgoBlob(task.ModelStart)
	if err != nil {
		return fmt.Errorf("Error pulling model %s from storage: %s", task.ModelStart, err)
	}
	defer model.Close()

	modelImageName := fmt.Sprintf("%s-%s", w.modelImagePrefix, task.ModelStart)
	err = w.ModelImageLoad(modelImageName, model)
	if err != nil {
		return fmt.Errorf("Error loading model image %s in Docker daemon: %s", modelImageName, err)
	}
	defer w.containerRuntime.ImageUnload(modelImageName)

	// Setup directory structure
	taskDataFolder := filepath.Join(w.dataFolder, task.ModelStart.String())
	trainFolder := filepath.Join(taskDataFolder, w.trainFolder)
	testFolder := filepath.Join(taskDataFolder, w.testFolder)
	untargetedTestFolder := filepath.Join(taskDataFolder, w.untargetedTestFolder)
	modelFolder := filepath.Join(taskDataFolder, w.modelFolder)
	err = os.MkdirAll(trainFolder, os.ModeDir)
	if err != nil {
		return fmt.Errorf("Error creating train folder under %s: %s", trainFolder, err)
	}
	err = os.MkdirAll(testFolder, os.ModeDir)
	if err != nil {
		return fmt.Errorf("Error creating test folder under %s: %s", testFolder, err)
	}
	err = os.MkdirAll(untargetedTestFolder, os.ModeDir)
	if err != nil {
		return fmt.Errorf("Error creating untargeted test folder under %s: %s", untargetedTestFolder, err)
	}
	err = os.MkdirAll(modelFolder, os.ModeDir)
	if err != nil {
		return fmt.Errorf("Error creating model folder under %s: %s", untargetedTestFolder, err)
	}

	// Let's make sure these folders are wiped out once the task is done/failed
	defer os.RemoveAll(taskDataFolder)

	// Pulling train dataset
	for _, dataID := range task.TrainData {
		data, err := w.storage.GetDataBlob(dataID)
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
		data, err := w.storage.GetDataBlob(dataID)
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
	_, err = w.UntargetTestingVolume(problemImageName, testFolder, untargetedTestFolder)
	if err != nil {
		return fmt.Errorf("Error preparing problem %s for %s: %s", task.Problem, task.ModelStart, err)
	}

	// Let's pass the task to our execution backend, now that everything should be in place
	_, err = w.Train(modelImageName, trainFolder, untargetedTestFolder, modelFolder)
	if err != nil {
		return fmt.Errorf("Error in train task: %s -- Body: %s", err, task)
	}

	// Let's move test predictions to the test folder with targets
	os.Rename(
		fmt.Sprintf("%s/%s", untargetedTestFolder, w.predFolder),
		fmt.Sprintf("%s/%s", testFolder, w.predFolder),
	)

	// Let's compute the performance !
	_, err = w.ComputePerf(problemImageName, trainFolder, testFolder)
	if err != nil {
		// FIXME: do not return here
		return fmt.Errorf("Error computing perf for problem %s and model (new) %s: %s", task.Problem, task.ModelEnd, err)
	}

	// Let's create a new model and post it to storage TODO change that: models should be tar.gz files
	// of the model volume after the training
	// newModelImageName := fmt.Sprintf("%s-%s", w.modelImagePrefix, task.ModelEnd)
	// snapshot, err := w.containerRuntime.SnapshotContainer(containerID, newModelImageName)
	// if err != nil {
	// 	return fmt.Errorf("Error snapshotting container %s to image %s: %s", containerID, newModelImageName, err)
	// }

	// err = w.storage.PostAlgo(task.ModelEnd, snapshot)
	// if err != nil {
	// 	return fmt.Errorf("Error streaming new model %s to storage: %s", task.ModelEnd, err)
	// }

	// Let's send the perf file to the orchestrator
	performanceFilePath := fmt.Sprintf("%s/performance.json", testFolder)
	resultFile, err := os.Open(performanceFilePath)
	if err != nil {
		return fmt.Errorf("Error reading performance file %s: %s", performanceFilePath, err)
	}
	defer resultFile.Close()

	err = w.orchestrator.PostLearnResult(task.ID, resultFile)
	if err != nil {
		return fmt.Errorf("Error posting learn result %s to orchestrator: %s", task.ModelEnd, err)
	}

	log.Printf("[INFO] Train finished with success, cleaning up...")

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
//	_, err = w.Predict(modelImageName, untargetedTestFolder)
//	if err != nil {
//		return fmt.Errorf("Error in test task: %s -- Body: %s", err, task)
//	}
//
// 	// TODO: send the prediction to the viewer, asynchronously
// 	log.Printf("Predicition completed with success. Predicition %s", prediction)
//
// 	return
// }

// ProblemWorkflowImageLoad loads the docker image corresponding to a problem workflow in the Docker
// daemon that will then run this problem workflow
func (w *Worker) ProblemWorkflowImageLoad(problemImage string, imageReader io.Reader) error {
	imageTarReader, err := gzip.NewReader(imageReader)
	if err != nil {
		return fmt.Errorf("Error un-gzipping problem workflow image %s: %s", problemImage, err)
	}
	// defer imageTarReader.Close()

	image, err := w.containerRuntime.ImageBuild(problemImage, imageTarReader)
	if err != nil {
		return fmt.Errorf("Error building problem workflow image %s: %s", problemImage, err)
	}
	// defer image.Close()

	return w.containerRuntime.ImageLoad(problemImage, image)
}

// ModelImageLoad loads the Docker image corresponding to a given model
func (w *Worker) ModelImageLoad(modelImage string, imageReader io.Reader) error {
	imageTarReader, err := gzip.NewReader(imageReader)
	if err != nil {
		return fmt.Errorf("Error un-gzipping model image %s: %s", modelImage, err)
	}
	defer imageTarReader.Close()

	image, err := w.containerRuntime.ImageBuild(modelImage, imageTarReader)
	if err != nil {
		return fmt.Errorf("Error building model image %s: %s", modelImage, err)
	}
	defer image.Close()

	return w.containerRuntime.ImageLoad(modelImage, image)
}

// UntargetTestingVolume copies data from /<host-data-volume>/<model>/data to
// /<host-data-volume>/<model>/train and removes targets from test files... using the problem
// workflow container.
func (w *Worker) UntargetTestingVolume(problemImage, testFolder, untargetedTestFolder string) (containerID string, err error) {
	return w.containerRuntime.RunImageInUntrustedContainer(
		problemImage,
		[]string{"-T", "detarget", "-i", "/true_data", "-s", "/pred_data"},
		map[string]string{
			testFolder:           "/true_data/test",
			untargetedTestFolder: "/pred_data/test",
		}, true)
}

// Train launches the submission container's train routines
func (w *Worker) Train(modelImage, trainFolder, testFolder, modelFolder string) (containerID string, err error) {
	return w.containerRuntime.RunImageInUntrustedContainer(
		modelImage,
		[]string{"-V", "/data", "-T", "train"},
		map[string]string{
			trainFolder: "/data/train",
			testFolder:  "/data/test",
			modelFolder: "/data/model",
		}, false)
}

// Predict launches the submission container's predict routines
func (w *Worker) Predict(modelImage, testFolder string) (containerID string, err error) {
	return w.containerRuntime.RunImageInUntrustedContainer(
		modelImage,
		[]string{"-V", "/data", "-T", "predict"},
		map[string]string{
			testFolder: "/true_data",
		}, true)
}

// ComputePerf analyses the prediction folders and computes a score for the model
func (w *Worker) ComputePerf(problemImage, trainFolder, testFolder string) (containerID string, err error) {
	return w.containerRuntime.RunImageInUntrustedContainer(
		problemImage,
		[]string{"-T", "perf", "-i", "/true_data", "-s", "/pred_data"},
		map[string]string{
			testFolder:  "/true_data/test",
			trainFolder: "/pred_data/train",
		}, true)
}

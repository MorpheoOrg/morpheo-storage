package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/MorpheoOrg/morpheo-compute/common"
)

// Worker stores references to our chosen backends
type Worker struct {
	executionBackend common.ExecutionBackend
	storageBackend   common.StorageBackend
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

	// Let's pass the learn task to our execution backend
	score, err := w.executionBackend.Train(task.Algo, task.TrainData, task.TestData)
	if err != nil {
		return fmt.Errorf("Error in train task: %s -- Body: %s", err, message)
	}

	// TODO: update the score (notify the orchestrator ?)
	log.Printf("Train finished with success. Score %f", score)

	return
}

// HandlePred handles our prediction tasks
func (w *Worker) HandlePred(message []byte) (err error) {
	var task common.Preduplet
	err = json.NewDecoder(bytes.NewReader(message)).Decode(&task)
	if err != nil {
		return fmt.Errorf("Error un-marshaling pred-uplet: %s -- Body: %s", err, message)
	}

	// Let's pass the prediction task to our execution backend
	prediction, err := w.executionBackend.Predict(task.Model, task.Data)
	if err != nil {
		return fmt.Errorf("Error in prediction task: %s -- Body: %s", err, message)
	}

	// TODO: send the prediction to the viewer, asynchronously
	log.Printf("Predicition completed with success. Predicition %s", prediction)

	return
}

func main() {
	// TODO: improve config and add a -container-backend flag and relevant opts
	// TODO: add NSQ consumer flags
	var (
		lookupUrls           common.MultiStringFlag
		topic                string
		channel              string
		queuePollingInterval time.Duration
	)

	flag.Var(&lookupUrls, "lookup-urls", "The URLs of the Nsqlookupd instances to fetch our topics from.")
	flag.StringVar(&topic, "topic", "learn", "The topic of the Nsqd/Nsqlookupd instance to listen to.")
	flag.StringVar(&channel, "channel", "compute", "The channel to use (default: compute)")
	flag.DurationVar(&queuePollingInterval, "lookup-interval", 5*time.Second, "The interval at which nsqlookupd will be polled")
	flag.Parse()

	// Config check
	if len(lookupUrls) == 0 {
		lookupUrls = append(lookupUrls, "nsqlookupd:6460")
	}

	if topic != common.TrainTopic && topic != common.PredictTopic {
		log.Panicf("Unknown topic: %s, valid values are %s and %s", topic, common.TrainTopic, common.PredictTopic)
	}

	// Let's connect with Storage (TODO: replace our mock with the real storage)
	storageBackend := common.NewStorageAPIMock()

	// Let's hook to our container backend and create a Worker instance containing
	// our message handlers TODO: put data folders in flags
	executionBackend, err := common.NewDockerBackend("/data")
	if err != nil {
		log.Panicf("Impossible to connect to Docker container backend: %s", err)
	}
	worker := Worker{
		executionBackend: executionBackend,
		storageBackend:   storageBackend,
	}

	// Let's hook with our consumer
	consumer := common.NewNSQConsumer(lookupUrls, channel, queuePollingInterval)

	// Wire our message handlers
	consumer.AddHandler(common.TrainTopic, worker.HandleLearn, 1)
	// consumer.AddHandler(common.PredictTopic, worker.HandlePred, 1)

	// Let's connect to the for real and start pulling tasks
	consumer.ConsumeUntilKilled()

	log.Println("Consumer has been gracefully stopped... Bye bye!")
	return
}

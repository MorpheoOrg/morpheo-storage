package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/DeepSee/dc-compute"
	common "github.com/DeepSee/dc-compute/common"
)

type Worker struct {
	backend common.ContainerBackend
}

func (w *Worker) HandleLearn(message []byte) (err error) {
	var task dccompute.LearnTask
	err = json.NewDecoder(bytes.NewReader(message)).Decode(&task)
	if err != nil {
		return common.NewHandlerFatalError(fmt.Errorf("Error un-marshaling train task: %s -- Body: %s", err, message))
	}

	if err = task.Check(); err != nil {
		return common.NewHandlerFatalError(fmt.Errorf("Error in train task: %s -- Body: %s", err, message))
	}

	// TODO: pull data from storage in the trusted container

	err = w.backend.RunInUntrustedContainer(task.LearnUplet.Model.String(), task.LearnUplet.Model.String(), []string{task.Data.String()}, time.Hour)
	return
}

func (w *Worker) HandleTest(message []byte) (err error) {
	var task dccompute.TestTask
	err = json.NewDecoder(bytes.NewReader(message)).Decode(&task)
	if err != nil {
		return fmt.Errorf("Error un-marshaling test task: %s -- Body: %s", err, message)
	}

	// TODO: pull data from storage in the trusted container

	err = w.backend.RunInUntrustedContainer(task.LearnUplet.Model.String(), task.LearnUplet.Model.String(), []string{task.Data.String()}, time.Hour)
	return
}

func (w *Worker) HandlePred(message []byte) (err error) {
	var task dccompute.Preduplet
	err = json.NewDecoder(bytes.NewReader(message)).Decode(&task)
	if err != nil {
		return fmt.Errorf("Error un-marshaling pred-uplet: %s -- Body: %s", err, message)
	}

	// TODO: pull data from storage in the trusted container

	err = w.backend.RunInUntrustedContainer(task.Model.String(), task.Model.String(), []string{task.Data.String()}, time.Hour)
	return
}

func main() {
	// TODO: improve config and add a -container-backend flag
	var (
		lookupUrls           dccompute.MultiStringFlag
		topic                string
		channel              string
		trustedImage         string
		queuePollingInterval time.Duration
	)

	flag.Var(&lookupUrls, "lookup-urls", "The URLs of the Nsqlookupd instances to fetch our topics from.")
	flag.StringVar(&topic, "topic", "learn", "The topic of the Nsqd/Nsqlookupd instance to listen to.")
	flag.StringVar(&channel, "channel", "compute", "The channel to use (default: compute)")
	flag.StringVar(&trustedImage, "trusted-container-image", "registry.localhost/compute-trusted-sidecar", "URL to pull the trusted container image from")
	flag.DurationVar(&queuePollingInterval, "lookup-interval", time.Second, "The interval at which nsqlookupd will be polled")
	flag.Parse()

	// Config check
	if len(lookupUrls) == 0 {
		lookupUrls = append(lookupUrls, "nsqlookupd:6460")
	}

	if topic != dccompute.LearnTopic && topic != dccompute.PredictionTopic && topic != dccompute.TestTopic {
		log.Panicf("Unknown topic: %s, valid values are %s, %s and %s", topic, dccompute.LearnTopic, dccompute.TestTopic, dccompute.PredictionTopic)
	}

	// Let's hook to our container backend and create a Worker instance containing
	// our message handlers
	containerBackend, err := common.NewDockerBackend("file://var/run/docker.sock", trustedImage)
	if err != nil {
		log.Panicf("Impossible to connect to Docker container backend: %s", err)
	}
	worker := Worker{
		backend: containerBackend,
	}

	// Let's hook with our consumer
	consumer := common.NewNSQConsumer(lookupUrls, channel, queuePollingInterval)

	// Wire our message handlers
	consumer.AddHandler("learn", worker.HandleLearn, 2)
	consumer.AddHandler("test", worker.HandleTest, 1)
	consumer.AddHandler("pred", worker.HandlePred, 1)

	// Let's connect to the for real and start pulling tasks
	consumer.ConsumeUntilKilled()

	log.Println("Consumer has been gracefully stopped... Bye bye!")
	return
}

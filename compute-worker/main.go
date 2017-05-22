package main

import (
	"log"
	"time"

	"github.com/MorpheoOrg/go-morpheo/client"
	"github.com/MorpheoOrg/go-morpheo/common"
)

func main() {
	conf := NewConsumerConfig()

	// Let's connect with Storage (or use our mock if no storage host was provided)
	var storageBackend client.Storage
	// if conf.StorageHost != "" {
	storageBackend = &client.StorageAPI{
		Hostname: conf.StorageHost,
		Port:     conf.StoragePort,
	}
	// } else {
	// 	storageBackend = client.NewStorageAPIMock()
	// }

	// And with the orchestrator
	var orchestratorBackend client.Orchestrator
	if conf.OrchestratorHost != "" {
		orchestratorBackend = &client.OrchestratorAPI{
			Hostname: conf.OrchestratorHost,
			Port:     conf.OrchestratorPort,
		}
	} else {
		orchestratorBackend = client.NewOrchestratorAPIMock()
	}

	// Let's hook to our container backend and create a Worker instance containing
	// our message handlers
	containerRuntime, err := common.NewDockerRuntime(conf.DockerTimeout)
	if err != nil {
		log.Panicf("[FATAL ERROR] Impossible to connect to Docker container backend: %s", err)
	}

	worker := NewWorker(
		// Root folder for train/test/predict data (should shared with the container runtime)
		"/data",
		// Subfolder names
		"train",
		"test",
		"untargeted_test",
		"pred",
		"model",
		// Container runtime image name prefixes
		"problem",
		"algo",
		// Dependency injection is done here :)
		containerRuntime,
		storageBackend,
		orchestratorBackend,
	)

	// Let's hook with our consumer
	consumer := common.NewNSQConsumer(conf.NsqlookupdURLs, "compute", 5*time.Second)

	// Wire our message handlers
	consumer.AddHandler(common.TrainTopic, worker.HandleLearn, conf.LearnParallelism, conf.LearnTimeout)
	// TODO: add the prediction handler too.
	// consumer.AddHandler(common.PredictTopic, worker.HandlePred, conf.PredictParallelism, conf.PredictTimeout)

	// Let's connect to the for real and start pulling tasks
	consumer.ConsumeUntilKilled()

	log.Println("[INFO] Consumer has been gracefully stopped... Bye bye!")
	return
}

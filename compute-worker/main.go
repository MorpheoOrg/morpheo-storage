package main

import (
	"flag"
	"log"
	"time"

	"github.com/MorpheoOrg/go-morpheo/client"
	"github.com/MorpheoOrg/go-morpheo/common"
)

func main() {
	// TODO: improve config and add a -container-backend flag and relevant opts
	// TODO: add NSQ consumer flags
	var (
		lookupUrls           common.MultiStringFlag
		channel              string
		queuePollingInterval time.Duration
	)

	flag.Var(&lookupUrls, "lookup-urls", "The URLs of the Nsqlookupd instances to fetch our topics from.")
	flag.StringVar(&channel, "channel", "compute", "The channel to use (default: compute)")
	flag.DurationVar(&queuePollingInterval, "lookup-interval", 5*time.Second, "The interval at which nsqlookupd will be polled")
	flag.Parse()

	// Config check
	if len(lookupUrls) == 0 {
		lookupUrls = append(lookupUrls, "nsqlookupd:6460")
	}

	// TODO: flags to choose backends or mocks

	// Let's connect with Storage (TODO: flags flags flags)
	storageBackend := &client.StorageAPI{
		Hostname: "storage",
		Port:     80,
	}

	// And with the orchestrator (TODO: flags flags flags)
	orchestratorBackend := &client.OrchestratorAPI{
		Hostname: "orchestrator",
		Port:     80,
	}

	// Let's hook to our container backend and create a Worker instance containing
	// our message handlers
	// TODO timeout in a flag
	containerRuntime, err := common.NewDockerRuntime(600 * time.Second)
	if err != nil {
		log.Panicf("[FATAL ERROR] Impossible to connect to Docker container backend: %s", err)
	}

	// TODO: put these arguments in flags
	worker := NewWorker(
		"/data",
		"train",
		"test",
		"untargeted_test",
		"pred",
		"model",
		"problem",
		"test",
		containerRuntime,
		storageBackend,
		orchestratorBackend,
	)

	// Let's hook with our consumer
	consumer := common.NewNSQConsumer(lookupUrls, channel, queuePollingInterval)

	// Wire our message handlers
	consumer.AddHandler(common.TrainTopic, worker.HandleLearn, 1)
	// TODO: add the prediction handler too.
	// consumer.AddHandler(common.PredictTopic, worker.HandlePred, 1)

	// Let's connect to the for real and start pulling tasks
	consumer.ConsumeUntilKilled()

	log.Println("[INFO] Consumer has been gracefully stopped... Bye bye!")
	return
}

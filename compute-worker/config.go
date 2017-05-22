package main

import (
	"flag"
	"time"

	"github.com/MorpheoOrg/go-morpheo/common"
)

// ConsumerConfig holds the consumer configuration
type ConsumerConfig struct {
	// Broker
	NsqlookupdURLs     []string
	LearnParallelism   int
	PredictParallelism int
	LearnTimeout       time.Duration
	PredictTimeout     time.Duration

	// Other compute services
	OrchestratorHost string
	OrchestratorPort int
	StorageHost      string
	StoragePort      int

	// Container Runtime
	DockerHost    string
	DockerTimeout time.Duration
}

// NewConsumerConfig parses CLI flags, generates and validates a ConsumerConfig
func NewConsumerConfig() (conf *ConsumerConfig) {
	var (
		nsqlookupdURLs     common.MultiStringFlag
		learnParallelism   int
		predictParallelism int
		learnTimeout       time.Duration
		predictTimeout     time.Duration

		orchestratorHost string
		orchestratorPort int
		storageHost      string
		storagePort      int

		dockerHost    string
		dockerTimeout time.Duration
	)

	// CLI Flags
	flag.Var(&nsqlookupdURLs, "nsqlookupd-urls", "URL(s) of NSQLookupd instances to connect to")
	flag.IntVar(&learnParallelism, "learn-parallelism", 1, "Number of learning task that this worker can execute in parallel.")
	flag.IntVar(&predictParallelism, "predict-parallelism", 1, "Number of learning task that this worker can execute in parallel.")
	flag.DurationVar(&learnTimeout, "learn-timeout", 20*time.Minute, "After this delay, learning tasks are timed out (default: 20m)")
	flag.DurationVar(&predictTimeout, "predict-timeout", 20*time.Minute, "After this delay, prediction tasks are timed out (default: 20m)")

	flag.StringVar(&orchestratorHost, "orchestrator-host", "", "Hostname of the orchestrator to send notifications to (leave blank to use the Orchestrator API Mock)")
	flag.IntVar(&orchestratorPort, "orchestrator-port", 80, "TCP port to contact the orchestrator on (default: 80)")

	flag.StringVar(&storageHost, "storage-host", "", "Hostname of the storage API to retrieve data frome (leave blank to use the Storage API Mock)")
	flag.IntVar(&storagePort, "storage-port", 80, "TCP port to contact storage on (default: 80)")

	flag.StringVar(&dockerHost, "docker-host", "unix://var/run/docker.sock", "URI of the Docker daemon to run containers")

	flag.DurationVar(&dockerTimeout, "docker-timeout", 15*time.Minute, "Docker commands timeout (concerns builds, runs, pulls, etc...) (default: 15m)")

	flag.Parse()

	if len(nsqlookupdURLs) == 0 {
		nsqlookupdURLs = append(nsqlookupdURLs, "nsqlookupd:4161")
	}

	return &ConsumerConfig{
		NsqlookupdURLs:     nsqlookupdURLs,
		LearnParallelism:   learnParallelism,
		PredictParallelism: predictParallelism,
		LearnTimeout:       learnTimeout,
		PredictTimeout:     predictTimeout,

		// Other compute services
		OrchestratorHost: orchestratorHost,
		OrchestratorPort: orchestratorPort,
		StorageHost:      storageHost,
		StoragePort:      storagePort,

		// Container Runtime
		DockerHost:    dockerHost,
		DockerTimeout: dockerTimeout,
	}
}

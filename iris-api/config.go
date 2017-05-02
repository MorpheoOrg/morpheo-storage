package main

import (
	"flag"
	"sync"

	"github.com/MorpheoOrg/morpheo-compute/common"
)

// Consumer topics: names of the different work queues
const (
	PredictionTopic = "prediction"
	TestTopic       = "test"
	LearnTopic      = "train"
)

// ProducerConfig Compute API configuration, subject to dynamic changes for the addresses of
// storage & orchestrator endpoints, and any RESTFul HTTP API added in the future.
type ProducerConfig struct {
	Hostname             string
	Port                 int
	OchestratorEndpoints []string
	StorageEndpoints     []string
	Broker               string
	BrokerHost           string
	BrokerPort           int
	CertFile             string
	KeyFile              string

	lock sync.Mutex
}

// TLSOn Returns true if TLS credentials have been provided
func (c *ProducerConfig) TLSOn() bool {
	return c.CertFile != "" && c.KeyFile != ""
}

// Locks the config store
func (c *ProducerConfig) Lock() {
	c.lock.Lock()
}

// Allows the config store to be written to
func (c *ProducerConfig) Unlock() {
	c.lock.Unlock()
}

// Compute the configuration object. Note that a pointer is returned not to
// avoid copy but rather to allow the configuration to be dynamically changed.
// If this isn't possible with a flags or env. variables, we may later make it
// possible to get the config from a K/V store such as etcd or consul to allow
// dynamic conf updates without requiring a restart.
//
// When using the config, please keep in mind that it can therefore be changed
// at any time. If you don't want this to happen, please use the object's
// Lock()/Unlock() features.
func NewProducerConfig() (conf *ProducerConfig) {
	var (
		hostname      string
		port          int
		orchestrators common.MultiStringFlag
		storages      common.MultiStringFlag
		broker        string
		brokerHost    string
		brokerPort    int
		certFile      string
		keyFile       string
	)

	// CLI Flags
	flag.StringVar(&hostname, "host", "0.0.0.0", "The hostname our server will be listening on")
	flag.IntVar(&port, "port", 8000, "The port our compute API will be listening on")
	flag.Var(&orchestrators, "orchestrator", "List of endpoints (scheme and port included) for the orchestrators we want to bind to.")
	flag.Var(&storages, "storage", "List of endpoints (scheme and port included) for the storage nodes to bind to.")
	flag.StringVar(&broker, "broker", common.BrokerNSQ, "Broker type to use (only 'nsq' available for now)")
	flag.StringVar(&brokerHost, "broker-host", "nsqd", "The address of the NSQ Broker to talk to")
	flag.IntVar(&brokerPort, "broker-port", 4160, "The port of the NSQ Broker to talk to")
	flag.StringVar(&certFile, "cert", "", "The ")
	flag.StringVar(&keyFile, "key", "", "The address of the NSQ Broker to talk to")
	flag.Parse()

	// Apply custom defaults on list flags if necessary
	if len(orchestrators) == 0 {
		orchestrators = append(orchestrators, "http://orchestrator")
	}

	if len(storages) == 0 {
		storages = append(storages, "http://storages")
	}

	// Let's create the config structure
	conf = &ProducerConfig{
		Hostname:             hostname,
		Port:                 port,
		OchestratorEndpoints: orchestrators,
		StorageEndpoints:     storages,
		Broker:               broker,
		BrokerHost:           brokerHost,
		BrokerPort:           brokerPort,
		CertFile:             certFile,
		KeyFile:              keyFile,
	}
	return
}

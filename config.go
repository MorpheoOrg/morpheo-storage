package dccompute

import (
	"flag"
	"strings"
	"sync"
)

// DC-Compute configuration, subject to dynamic changes for the addresses of
// storage & orchestrator endpoints, and any RESTFul HTTP API added in the
// future.
type Config struct {
	Hostname             string
	Port                 int
	OchestratorEndpoints []string
	StorageEndpoints     []string
	BrokerHost           string
	BrokerPort           int
	CertFile             string
	KeyFile              string

	lock sync.Mutex
}

// Returns true if TLS credentials have been provided
func (c *Config) TLSOn() bool {
	return c.CertFile != "" && c.KeyFile != ""
}

func (c *Config) Lock() {
	c.lock.Lock()
}

func (c *Config) Unlock() {
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
func NewConfig() (conf *Config) {
	var (
		hostname      string
		port          int
		orchestrators MultiStringFlag
		storages      MultiStringFlag
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
	flag.StringVar(&brokerHost, "broker", "localhost", "The address of the NSQ Broker to talk to")
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
	conf = &Config{
		Hostname:             hostname,
		Port:                 port,
		OchestratorEndpoints: orchestrators,
		StorageEndpoints:     storages,
		BrokerHost:           brokerHost,
		BrokerPort:           brokerPort,
		CertFile:             certFile,
		KeyFile:              keyFile,
	}
	return
}

// MultiStringFlag is a flag for passing multiple parameters using same flag
type MultiStringFlag []string

// String returns string representation of the node groups.
func (flag *MultiStringFlag) String() string {
	return "[" + strings.Join(*flag, " ") + "]"
}

// Set adds a new configuration.
func (flag *MultiStringFlag) Set(value string) error {
	*flag = append(*flag, value)
	return nil
}

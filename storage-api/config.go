package main

import "flag"

// StorageConfig holds the configuration variables for the storage API
type StorageConfig struct {
	Hostname string
	Port     int
	CertFile string
	KeyFile  string
}

// TLSOn returns true if TLS credentials have been provided. The API will then
// serve requests over TLS.
func (c *StorageConfig) TLSOn() bool {
	return c.CertFile != "" && c.KeyFile != ""
}

// NewStorageConfig computes the configuration object parsing CLI flags
func NewStorageConfig() (conf *StorageConfig) {
	var (
		hostname string
		port     int
		certFile string
		keyFile  string
	)

	// CLI Flags
	flag.StringVar(&hostname, "host", "0.0.0.0", "The hostname our server will be listening on")
	flag.IntVar(&port, "port", 8000, "The port our compute API will be listening on")
	flag.StringVar(&certFile, "cert", "", "The TLS certs to serve to clients (leave blank for no TLS)")
	flag.StringVar(&keyFile, "key", "", "The TLS key used to encrypt connection (leave blank for no TLS)")
	flag.Parse()

	// Let's create the config structure
	conf = &StorageConfig{
		Hostname: hostname,
		Port:     port,
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	return
}

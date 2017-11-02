/*
 * Copyright Morpheo Org. 2017
 *
 * contact@morpheo.co
 *
 * This software is part of the Morpheo project, an open-source machine
 * learning platform.
 *
 * This software is governed by the CeCILL license, compatible with the
 * GNU GPL, under French law and abiding by the rules of distribution of
 * free software. You can  use, modify and/ or redistribute the software
 * under the terms of the CeCILL license as circulated by CEA, CNRS and
 * INRIA at the following URL "http://www.cecill.info".
 *
 * As a counterpart to the access to the source code and  rights to copy,
 * modify and redistribute granted by the license, users are provided only
 * with a limited warranty  and the software's author,  the holder of the
 * economic rights,  and the successive licensors  have only  limited
 * liability.
 *
 * In this respect, the user's attention is drawn to the risks associated
 * with loading,  using,  modifying and/or developing or reproducing the
 * software by the user in light of its specific status of free software,
 * that may mean  that it is complicated to manipulate,  and  that  also
 * therefore means  that it is reserved for developers  and  experienced
 * professionals having in-depth computer knowledge. Users are therefore
 * encouraged to load and test the software's suitability as regards their
 * requirements in conditions enabling the security of their systems and/or
 * data to be ensured and,  more generally, to use and operate it in the
 * same conditions as regards security.
 *
 * The fact that you are presently reading this means that you have had
 * knowledge of the CeCILL license and that you accept its terms.
 */

package main

import "flag"

// StorageConfig holds the configuration variables for the storage API
type StorageConfig struct {
	// API Server settings
	Hostname string
	Port     int
	CertFile string
	KeyFile  string

	// Authentification
	APIUser     string
	APIPassword string

	// Database configuration
	DBHost string
	DBPort int
	DBUser string
	DBPass string
	DBName string

	// Database migration flags
	DBMigrationsDir string
	DBRollback      bool

	// Blobstore
	BlobStore string

	// local (disk) blob store configuration
	DataDir string
	// S3 config
	AWSBucket string
	AWSRegion string
	// Google Cloud Config
	GCBucket string
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

		apiUser     string
		apiPassword string

		dbHost string
		dbPort int
		dbUser string
		dbPass string
		dbName string

		dbMigrationsDir string
		dbRollback      bool

		blobStore string

		dataDir   string
		awsBucket string
		awsRegion string
		gcBucket  string
	)

	// CLI Flags
	flag.StringVar(&hostname, "host", "0.0.0.0", "The hostname our server will be listening on")
	flag.IntVar(&port, "port", 8000, "The port our compute API will be listening on")
	flag.StringVar(&certFile, "cert", "", "The TLS certs to serve to clients (leave blank for no TLS)")
	flag.StringVar(&keyFile, "key", "", "The TLS key used to encrypt connection (leave blank for no TLS)")

	flag.StringVar(&apiUser, "user", "u", "The username for Basic Authentification")
	flag.StringVar(&apiPassword, "password", "p", "The password for Basic Authentification")

	flag.StringVar(&dbHost, "db-host", "postgres", "The hostname of the postgres database (default: postgres)")
	flag.IntVar(&dbPort, "db-port", 5432, "The database port")
	flag.StringVar(&dbName, "db-name", "db", "The database name (default: morpheo_storage)")
	flag.StringVar(&dbUser, "db-user", "u", "The database user (default: storage)")
	flag.StringVar(&dbPass, "db-pass", "p", "The database password to use (default: tooshort)")

	flag.StringVar(&dbMigrationsDir, "db-migrations-dir", "/migrations", "The database migrations directory (default: /migrations)")
	flag.BoolVar(&dbRollback, "db-rollback", false, "if true, rolls back the last migration (default: false)")

	flag.StringVar(&blobStore, "blobstore", "local", "Storage service provider: 'gc' for Google Cloud Storage, 's3' for AWS S3, 'local' (default) and 'mock' supported")

	flag.StringVar(&dataDir, "data-dir", "/data", "The directory to store locally blob data under (default: /data)")
	flag.StringVar(&awsBucket, "s3-bucket", "", "AWS Bucket (default: empty string)")
	flag.StringVar(&awsRegion, "s3-region", "", "AWS Region (default: empty string)")
	flag.StringVar(&gcBucket, "gc-bucket", "", "Google Cloud Storage Bucket (default: empty string)")

	flag.Parse()

	// Let's create the config structure
	conf = &StorageConfig{
		Hostname: hostname,
		Port:     port,
		CertFile: certFile,
		KeyFile:  keyFile,

		APIUser:     apiUser,
		APIPassword: apiPassword,

		DBHost: dbHost,
		DBPort: dbPort,
		DBUser: dbUser,
		DBPass: dbPass,
		DBName: dbName,

		DBMigrationsDir: dbMigrationsDir,
		DBRollback:      dbRollback,

		BlobStore: blobStore,

		DataDir:   dataDir,
		AWSBucket: awsBucket,
		AWSRegion: awsRegion,
		GCBucket:  gcBucket,
	}
	return
}

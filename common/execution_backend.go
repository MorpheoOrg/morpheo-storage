package dccommon

import (
	uuid "github.com/satori/go.uuid"
)

// Abstraction for an execution backend (direct access to Docker/Rkt runtime or
// use of Swarm/Kubernetes/Mesos/Nomad/AWS Directly...)
//
// This interface should be abstract enough to allow non container backends
// (like EC2 VMs sharing an EBS volume). However we're only planning on using
// container based backends (Docker, Rkt and Kubernetes) for now.
type ExecutionBackend interface {
	Train(storage StorageBackend, modelID uuid.UUID, dataID uuid.UUID) (score float64, err error)
	Test(storage StorageBackend, modelID uuid.UUID, dataID uuid.UUID) (score float64, err error)
	Predict(storage StorageBackend, modelID uuid.UUID, dataID uuid.UUID) (prediction []byte, err error)
}

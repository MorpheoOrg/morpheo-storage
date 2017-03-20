package dccommon

// Abstraction for a Container backend (direct access to Docker/Rkt runtime or
// use of Swarm/Kubernetes/Mesos/Nomad...)
type ContainerBackend interface {
	RunInUntrustedContainer(imageName, action, data string) (err error)
}

package dccommon

// Abstraction for a Container backend (direct access to Docker/Rkt runtime or
// use of Swarm/Kubernetes/Mesos/Nomad...)
//
// More specifically, untrusted containers are always run with a "trusted"
// sidecar container with network access to download data from storage.
// This container shares a volume mounted as "/data" on the untrusted
// container. By any means, the untrusted container should have no network
// access (and all other unnecessary capabilities should be dropped). Of
// course, it goes without saying that commands inside the latter will be run
// by a user with limited priviledges (not root :) ).
type ContainerBackend interface {
	// Runs the provided command in the trusted container
	RunInTrustedContainer(command []string) (err error)

	// Runs the provided command in the untrusted container.
	// The container id to run the command in is used to fetch the container
	// from storage. Note that we leave the image author the possibility to
	// override the entrypoint af the Docker image.
	RunInUntrustedContainer(containerID string, args []string) (err error)
}

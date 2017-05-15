package common

import "io"

// ContainerRuntime abstracts Docker/rkt/... it can load/unload images and run them, in a secured
// way :)
type ContainerRuntime interface {
	// ImageBuildAndLoad builds an Image from a reader on a tar.gz archive containing all requirements
	// to build the image. In returns an io.ReadCloser on the image and an error if error there is.
	ImageBuild(name string, buildContext io.Reader) (image io.ReadCloser, err error)

	// ImageLoad loads a saved image from an io.Reader into the container runtime
	ImageLoad(name string, imageReader io.Reader) error

	// ImageUnload removes an Image from the ContainerRuntime's image store (aka from disk)
	ImageUnload(name string) error

	// Runs a given command in a network isolated container
	RunImageInUntrustedContainer(imageName string, args []string, mounts map[string]string, autoRemove bool) (containerID string, err error)

	// SnapshotContainer gets a snapshot of a given container and returns a ReadCloser on it.
	//
	// Note that it is up to the caller to call Close on the returned ReadCloser
	SnapshotContainer(containerID, imageName string) (image io.ReadCloser, err error)
}

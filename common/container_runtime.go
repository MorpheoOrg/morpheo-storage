package common

import "io"

// ContainerRuntime abstracts Docker/rkt/... it can load/unload images and run them, in a secured
// way :)
type ContainerRuntime interface {
	ImageLoad(name string, imageReader io.Reader) error
	ImageUnload(name string) error

	RunImageInUntrustedContainer(imageName string, args []string, mounts map[string]string, autoRemove bool) (containerID string, err error)
	SnapshotContainer(containerID, imageName string) (image io.ReadCloser, err error)
}

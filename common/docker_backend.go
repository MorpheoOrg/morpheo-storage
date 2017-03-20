package dccommon

import "fmt"

type DockerBackend struct {
	daemonURI string
}

func NewDockerBackend(daemonURI string) (b *DockerBackend, err error) {
	return &DockerBackend{
		daemonURI: daemonURI,
	}, nil
}

func (b *DockerBackend) RunInUntrustedContainer(imageName, action, data string) error {
	// TODO: implement this backend
	// TODO: pull the image/algorithm
	// TODO: pull the data
	// TODO: run the ML task in the container
	fmt.Printf("[docker-backend][not-implemented] Running %s task on data %s using model %s", action, data, imageName)
	return nil
}

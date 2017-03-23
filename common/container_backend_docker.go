package dccommon

import "fmt"

type DockerBackend struct {
	ContainerBackend

	daemonURI             string
	trustedContainerImage string
}

func NewDockerBackend(daemonURI string, trustedContainerImage string) (b *DockerBackend, err error) {
	return &DockerBackend{
		daemonURI:             daemonURI,
		trustedContainerImage: trustedContainerImage,
	}, nil
}

func (b *DockerBackend) RunInTrustedContainer(command []string) error {
	// TODO: implement this backend
	// TODO: pull the image/algorithm
	// TODO: pull the data
	// TODO: run the ML task in the container
	fmt.Printf("[docker-backend][not-implemented] Running `%s` in trusted container %s", command, b.trustedContainerImage)
	return nil
}

func (b *DockerBackend) RunInUntrustedContainer(containerID string, args []string) error {
	// TODO: implement this backend
	// TODO: pull the image/algorithm
	// TODO: pull the data
	// TODO: run the ML task in the container
	fmt.Printf("[docker-backend][not-implemented] Running `%s` in untrusted container %s", args, containerID)
	return nil
}

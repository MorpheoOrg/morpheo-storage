package dccommon

import (
	"context"
	"fmt"
	"log"
	"time"

	dockerContainer "github.com/docker/docker/api/types/container"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerCli "github.com/docker/docker/client"
)

type DockerBackend struct {
	ContainerBackend

	daemonURI             string
	trustedContainerImage string
	cli                   *dockerCli.Client
}

func NewDockerBackend(daemonURI string, trustedContainerImage string) (b *DockerBackend, err error) {
	apiClient, err := dockerCli.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Error creating Docker client: %s", err)
	}
	return &DockerBackend{
		daemonURI:             daemonURI,
		trustedContainerImage: trustedContainerImage,
		cli: apiClient,
	}, nil
}

func (b *DockerBackend) RunInTrustedContainer(containerName string, command []string, timeout time.Duration) error {
	log.Printf("[INFO][docker-backend] Running `%s` in trusted container %s", command, b.trustedContainerImage)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Let's create the container and run the command in it
	containerCreateBody, err := b.cli.ContainerCreate(
		ctx,
		&dockerContainer.Config{
			Hostname:     containerName,
			Domainname:   "",
			User:         "root:root",
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
			OpenStdin:    false,
			Env:          []string{},
			Cmd:          command,
			Entrypoint:   []string{},
			// TODO: attach a health check ?
			Image: b.trustedContainerImage,
			// TODO: volumes
			WorkingDir:      "/",
			NetworkDisabled: false,
			Labels:          map[string]string{},
			// StopSignal:
			// StopTimeout:
			// Shell
		},
		&dockerContainer.HostConfig{
			AutoRemove: true,
			// TODO: more stuff here
		},
		&dockerNetwork.NetworkingConfig{
		// TODO: same over here
		},
		containerName,
	)
	if err != nil {
		return fmt.Errorf("Error creating Docker container %s: %s", containerName, err)
	}

	// Let's log any warning that was trigger
	for n, warning := range containerCreateBody.Warnings {
		log.Printf("[WARNING %d][docker-backend] Warning creating container: %s", n, warning)
	}

	// Let's wait for the command to be over
	status, err := b.cli.ContainerWait(ctx, containerCreateBody.ID)
	if err != nil {
		return fmt.Errorf("Error waiting for trusted container to exit: %s", err)
	}

	log.Printf("[INFO][docker-backend] Trusted container ran command, status code: %d", status)

	return nil
}

func (b *DockerBackend) RunInUntrustedContainer(containerName string, containerID string, args []string, timeout time.Duration) error {
	log.Printf("[INFO][docker-backend] Running `%s` in untrusted container %s", args, containerID)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Let's create the container and run the command in it
	containerCreateBody, err := b.cli.ContainerCreate(
		ctx,
		&dockerContainer.Config{
			Hostname: containerName,
			// Domainname:   "",
			User:         "reek:greyjoy", // <-- eheheheh
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
			OpenStdin:    false,
			Env:          []string{},
			Cmd:          args,
			// TODO: make sure not setting the entrypoint makes Docker use the one defined in the image
			// TODO: attach a health check ?
			Image: containerID,
			// TODO: volumes
			WorkingDir:      "/",
			NetworkDisabled: true,
			Labels:          map[string]string{},
			// StopSignal:
			// StopTimeout:
			// Shell
		},
		&dockerContainer.HostConfig{
			AutoRemove: true,
			// TODO: more stuff here
		},
		&dockerNetwork.NetworkingConfig{
		// TODO: same over here
		},
		containerName,
	)
	if err != nil {
		return fmt.Errorf("Error creating Docker container %s: %s", containerName, err)
	}

	// Let's log any warning that was trigger
	for n, warning := range containerCreateBody.Warnings {
		log.Printf("[WARNING %d][docker-backend] Warning creating container: %s", n, warning)
	}

	// Let's wait for the command to be over
	status, err := b.cli.ContainerWait(ctx, containerCreateBody.ID)
	if err != nil {
		return fmt.Errorf("Error waiting for untrusted container to exit: %s", err)
	}

	log.Printf("[INFO][docker-backend] Untrusted container ran command, status code: %d", status)

	return nil
}

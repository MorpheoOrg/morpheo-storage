// TODO: this should be split into a generic interface "ContainerBackend" and
// two implementations: one for Docker and a similar one for Rkt
package dccommon

import (
	"context"
	"fmt"
	"log"
	"time"

	dockerContainer "github.com/docker/docker/api/types/container"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerCli "github.com/docker/docker/client"
	uuid "github.com/satori/go.uuid"
)

type DockerBackend struct {
	ExecutionBackend

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

func (b *DockerBackend) Train(storage StorageBackend, modelID uuid.UUID, dataID uuid.UUID) (score float64, err error) {
	// TODO: implement that
	return 1.0, nil
}

func (b *DockerBackend) Test(storage StorageBackend, modelID uuid.UUID, dataID uuid.UUID) (score float64, err error) {
	// TODO: implement that
	return 1.0, nil
}

func (b *DockerBackend) Predict(storage StorageBackend, modelID uuid.UUID, dataID uuid.UUID) (prediction []byte, err error) {
	// TODO: implement that
	return []byte("Irma"), nil
}

// Loads the model's Docker image in a registry accessible by the storage backend container runtime
// (a local Docker registry here)
func (b *DockerBackend) LoadModelFromStorage(storage StorageBackend, modelID uuid.UUID) (err error) {
	// TODO: implement that

	return nil
}

func (b *DockerBackend) LoadDataFromStorage(storage StorageBackend, dataID uuid.UUID) (err error) {
	// TODO: implement that

	return nil
}

func (b *DockerBackend) RunInTrustedContainer(command []string, timeout time.Duration) error {
	log.Printf("[INFO][docker-backend] Running `%s` in trusted container %s", command, b.trustedContainerImage)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// TODO: figure out what the f*** we do with that
	containerName := "trustedPieceOfShit"

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

func (b *DockerBackend) RunInUntrustedContainer(modelID string, args []string, timeout time.Duration) error {
	log.Printf("[INFO][docker-backend] Running `%s` in untrusted container %s", args, modelID)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	imageName := fmt.Sprintf("modelRegistry/%s", modelID)
	containerName := fmt.Sprintf("model-%s", modelID)

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
			Image: imageName,
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

package common

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerNetwork "github.com/docker/docker/api/types/network"
	dockerCli "github.com/docker/docker/client"
	uuid "github.com/satori/go.uuid"
)

// DockerRuntime implements ExecutionBackend for Docker
type DockerRuntime struct {
	ContainerRuntime

	timeout time.Duration
	docker  *dockerCli.Client
}

// NewDockerRuntime creates a new Docker execution backend
func NewDockerRuntime(timeout time.Duration) (b *DockerRuntime, err error) {
	apiClient, err := dockerCli.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Error creating Docker client: %s", err)
	}

	return &DockerRuntime{
		timeout: timeout,

		docker: apiClient,
	}, nil
}

// ImageBuild builds a Docker image from a given build context. The context actually simply is a tar
// archive of a folder containing a Dockerfile and all the files required to build that Dockerfile.
//
// Note that it is up to the caller to call Close() on the returned io.ReadCloser()
func (r *DockerRuntime) ImageBuild(name string, buildContext io.Reader) (image io.ReadCloser, err error) {
	dockerImage, err := r.docker.ImageBuild(context.Background(), buildContext, dockerTypes.ImageBuildOptions{
		Tags:           []string{name},
		SuppressOutput: false,
		NoCache:        false,
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
		// TODO: investigate this a bit too
		// Isolation: "",
		// NetworkMode    string
	})
	if err != nil {
		return nil, fmt.Errorf("[docker-runtime] Error building image %s: %s", name, err)
	}
	return dockerImage.Body, nil
}

// ImageLoad loads an image from a file into the Docker daemon (equivalent to the "docker load"
// command)
func (r *DockerRuntime) ImageLoad(name string, imageReader io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// TODO do something with the response, like checking the name of the loaded image
	_, err := r.docker.ImageLoad(ctx, imageReader, false)
	if err != nil {
		return fmt.Errorf("[docker-runtime] Error loading image %s: %s", name, err)
	}
	return nil
}

// ImageUnload removes an image from the Docker daemon (equivalent to the "docker rmi" command)
func (r *DockerRuntime) ImageUnload(imageID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// TODO do something with the response, like checking the name of the loaded image
	_, err := r.docker.ImageRemove(ctx, imageID, dockerTypes.ImageRemoveOptions{
		Force:         true,
		PruneChildren: false,
	})
	if err != nil {
		return fmt.Errorf("[docker-runtime] Error removing image %s: %s", imageID, err)
	}
	return nil
}

// RunImageInUntrustedContainer launch a container on the bound docker host with as many
// restrictions as possibe for our use case.
func (r *DockerRuntime) RunImageInUntrustedContainer(imageName string, args []string, mounts map[string]string, autoRemove bool) (containerID string, err error) {
	containerName := uuid.NewV4().String()
	log.Printf("[INFO][docker-backend] Running `%s` in untrusted container %s (image: %s)", args, containerName, imageName)

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	binds := []string{}
	for hostPath, containerPath := range mounts {
		binds = append(binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	// Let's create the container and run the command in it
	containerCreateBody, err := r.docker.ContainerCreate(
		ctx,
		&dockerContainer.Config{
			// Hostname: containerName,
			// Domainname:   "",
			User:         "root:root", // <-- FIXME nope, no damn way this will run as root :)
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
			OpenStdin:    false,
			// Env:          []string{},
			Cmd: args,
			// TODO: make sure not setting the entrypoint makes Docker use the one defined in the image
			Image:           imageName,
			WorkingDir:      "/data",
			NetworkDisabled: true,
			Labels:          map[string]string{},
			// StopSignal:
			// StopTimeout:
			// Shell
		},
		&dockerContainer.HostConfig{
			AutoRemove: autoRemove,
			Privileged: false,
			Binds:      binds,
			// TODO: investigate all capabilites and set capadd/capdrops accordingly
		},
		&dockerNetwork.NetworkingConfig{
		// TODO: investigate this a bit too
		},
		containerName,
	)
	log.Print("[DEBUG][docker-backend] Docker container created")
	if err != nil {
		return "", fmt.Errorf("Error creating Docker container %s: %s", containerName, err)
	}

	// Let's log any warning that was trigger
	for n, warning := range containerCreateBody.Warnings {
		log.Printf("[WARNING %d][docker-backend] Warning creating container: %s", n, warning)
	}

	err = r.docker.ContainerStart(
		ctx,
		containerCreateBody.ID,
		dockerTypes.ContainerStartOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("Error starting Docker container %s: %s", containerName, err)
	}

	// Let's wait for the command to be over
	status, err := r.docker.ContainerWait(ctx, containerCreateBody.ID)
	if err != nil {
		return "", fmt.Errorf("Error waiting for untrusted container to exit: %s", err)
	}

	log.Printf("[INFO][docker-backend] Untrusted container ran command, status code: %d", status)

	return containerCreateBody.ID, nil
}

// SnapshotContainer exports the trained container and pipes it in an image builder that forwards
// back a reader on the image's bytes.
func (r *DockerRuntime) SnapshotContainer(containerID, imageName string) (image io.ReadCloser, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	container, err := r.docker.ContainerExport(ctx, containerID)
	defer container.Close()
	if err != nil {
		return nil, fmt.Errorf("[docker-runtime] Error exporting container %s (image: %s): %s", containerID, imageName, err)
	}

	image, err = r.docker.ImageImport(
		ctx,
		dockerTypes.ImageImportSource{
			Source:     container,
			SourceName: "-",
		},
		imageName,
		dockerTypes.ImageImportOptions{
			Tag: "test",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("[docker-runtime] Error importing container %s to image %s: %s", containerID, imageName, err)
	}

	return
}

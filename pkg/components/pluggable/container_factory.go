package pluggable

import (
	"context"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	"github.com/yourbase/narwhal"
	"os"
	"path"
)

const (
	DaprSocketComponentPathEnvKey = "DAPR_COMPONENT_SOCKET_PATH"
)

type ContainerFactory struct {
	Image          string
	Version        string
	HostSocketRoot string
}

func (f *ContainerFactory) ImageString() string {
	return fmt.Sprintf("%s:%s", f.Image, f.Version)
}

type ComponentContainer struct {
	Id             string
	HostSocketPath string
}

func (f *ContainerFactory) StartContainer(ctx context.Context) (*ComponentContainer, error) {
	client := narwhal.DockerClient()
	socketUUID := uuid.New()
	socketFile := fmt.Sprintf("%s.sock", socketUUID)
	hostSocketPath := path.Join(f.HostSocketRoot, socketFile)
	containerSocketRoot := path.Join("/daprsock")
	containerSocketPath := path.Join(containerSocketRoot, socketFile)

	environment := []string{
		fmt.Sprintf("%s=%s", DaprSocketComponentPathEnvKey, containerSocketPath),
	}

	mounts := []docker.HostMount{
		{
			Source: f.HostSocketRoot,
			Target: containerSocketRoot,
		},
	}

	// Create a container
	if containerID, err := narwhal.CreateContainer(ctx, client, os.Stdout, &narwhal.ContainerDefinition{
		Image:       f.ImageString(),
		Environment: environment,
		Mounts:      mounts,
	}); err != nil {
		return nil, fmt.Errorf("unable to create container: %v", err)
	} else {
		return &ComponentContainer{
			Id:             containerID,
			HostSocketPath: hostSocketPath,
		}, nil
	}

}

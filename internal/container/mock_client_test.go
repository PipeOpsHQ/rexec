package container

import (
	"context"
	"io"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// MockDockerClient implements client.CommonAPIClient for testing
type MockDockerClient struct {
	client.CommonAPIClient
	ContainerCreateFunc     func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, containerName string) (container.CreateResponse, error)
	ContainerStartFunc      func(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStopFunc       func(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemoveFunc     func(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerListFunc       func(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ContainerInspectFunc    func(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ImagePullFunc           func(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error)
	ImageInspectWithRawFunc func(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	NetworkListFunc         func(ctx context.Context, options network.ListOptions) ([]network.Inspect, error)
	NetworkCreateFunc       func(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error)
	NetworkInspectFunc      func(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error)
	ContainerExecCreateFunc func(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error)
	ContainerExecAttachFunc func(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerStatsFunc      func(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error)
}

func (m *MockDockerClient) ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error) {
	if m.ContainerExecCreateFunc != nil {
		return m.ContainerExecCreateFunc(ctx, container, config)
	}
	return types.IDResponse{ID: "test-exec-id"}, nil
}

func (m *MockDockerClient) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
	if m.ContainerExecAttachFunc != nil {
		return m.ContainerExecAttachFunc(ctx, execID, config)
	}

	// Return a valid HijackedResponse with a dummy connection to avoid panic on Close()
	client, _ := net.Pipe()
	return types.HijackedResponse{
		Conn: client,
	}, nil
}

func (m *MockDockerClient) ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
	if m.ContainerStatsFunc != nil {
		return m.ContainerStatsFunc(ctx, containerID, stream)
	}
	return container.StatsResponseReader{}, nil
}

func (m *MockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	if m.ContainerListFunc != nil {
		return m.ContainerListFunc(ctx, options)
	}
	return []types.Container{}, nil
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if m.ContainerInspectFunc != nil {
		return m.ContainerInspectFunc(ctx, containerID)
	}
	return types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			HostConfig: &container.HostConfig{},
		},
		Config: &container.Config{},
	}, nil
}

func (m *MockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, containerName string) (container.CreateResponse, error) {
	if m.ContainerCreateFunc != nil {
		return m.ContainerCreateFunc(ctx, config, hostConfig, networkingConfig, platform, containerName)
	}
	return container.CreateResponse{ID: "test-container-id"}, nil
}

func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	if m.ContainerStartFunc != nil {
		return m.ContainerStartFunc(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	if m.ContainerStopFunc != nil {
		return m.ContainerStopFunc(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	if m.ContainerRemoveFunc != nil {
		return m.ContainerRemoveFunc(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerClient) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	if m.ImagePullFunc != nil {
		return m.ImagePullFunc(ctx, ref, options)
	}
	return nil, nil
}

func (m *MockDockerClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	if m.ImageInspectWithRawFunc != nil {
		return m.ImageInspectWithRawFunc(ctx, imageID)
	}
	return types.ImageInspect{}, []byte{}, nil
}

func (m *MockDockerClient) NetworkInspect(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
	if m.NetworkInspectFunc != nil {
		return m.NetworkInspectFunc(ctx, networkID, options)
	}
	return network.Inspect{}, nil
}

func (m *MockDockerClient) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Inspect, error) {
	if m.NetworkListFunc != nil {
		return m.NetworkListFunc(ctx, options)
	}
	return []network.Inspect{}, nil
}

func (m *MockDockerClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	if m.NetworkCreateFunc != nil {
		return m.NetworkCreateFunc(ctx, name, options)
	}
	return network.CreateResponse{ID: "test-network-id"}, nil
}

func (m *MockDockerClient) Close() error {
	return nil
}

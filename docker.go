package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
)

const MSG_CONTAINER_NOT_RUNNING = "Não foi possível realizar a ação: \"%s\". O container não está em estado de execução"

type Container struct {
	sourceVolume, targetVolume, tag, username, ID string
	dockerClient                                  *client.Client
}

type ImagePullEvent struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}

func (self Container) createVolume(ctx context.Context) error {
	_, err := self.dockerClient.VolumeCreate(ctx, volume.VolumeCreateBody{Name: self.sourceVolume})
	return err
}

func (self Container) removeVolume(ctx context.Context) error {
	return self.dockerClient.VolumeRemove(ctx, self.sourceVolume, false)
}

func (self Container) getImage(imageName string) string {
	image := imageName
	if len(imageName) == 0 {
		image = fmt.Sprintf("%s:%s", self.sourceVolume, self.tag)
		if len(self.username) > 0 {
			image = fmt.Sprintf("%s/%s", self.username, image)
		}
	}
	return image
}

func (self Container) pull(ctx context.Context, imageName string) error {
	var err error
	if len(imageName) > 0 || len(self.username) > 0 {
		err = imagePullOutputFormat(self.dockerClient.ImagePull(ctx,
			self.getImage(imageName), types.ImagePullOptions{}))
	}
	return err
}

func (self Container) commit(ctx context.Context) error {
	var err error = errors.New(fmt.Sprintf(MSG_CONTAINER_NOT_RUNNING, "commit"))

	if len(self.ID) > 0 {
		image := self.getImage("")

		_, err = self.dockerClient.ContainerCommit(ctx, self.ID,
			types.ContainerCommitOptions{Reference: image})
	}

	return err
}

func (self *Container) run(ctx context.Context, imageName string, cmd []string) error {

	containerConfig, hostConfig := self.config(imageName, cmd)

	cont, err := self.dockerClient.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, "")

	if err == nil {
		self.ID = cont.ID
		err = self.dockerClient.ContainerStart(ctx, self.ID, types.ContainerStartOptions{})
		if err == nil {
			err = self.logs(ctx)
		}
	}

	return err
}

func (self Container) remove(ctx context.Context) error {
	var err error = errors.New(fmt.Sprintf(MSG_CONTAINER_NOT_RUNNING, "remove"))
	if len(self.ID) > 0 {
		err = self.dockerClient.ContainerRemove(ctx, self.ID, types.ContainerRemoveOptions{})
	}
	return err
}

func (self Container) config(imageName string, cmd []string) (container.Config, container.HostConfig) {

	return container.Config{
			Image: self.getImage(imageName),
			Cmd:   cmd,
		}, container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: self.sourceVolume,
					Target: self.targetVolume,
				},
			},
		}
}

func (self Container) logs(ctx context.Context) error {
	out, err := self.dockerClient.ContainerLogs(ctx, self.ID,
		types.ContainerLogsOptions{ShowStdout: true, Follow: true})

	if err == nil && out != nil {
		defer out.Close()
		stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	}

	return err
}

func imagePullOutputFormat(out io.ReadCloser, err error) error {
	if err == nil && out != nil {
		defer out.Close()
		events := json.NewDecoder(out)

		var event *ImagePullEvent
		for {

			if err := events.Decode(&event); err != nil {
				if err == io.EOF {
					break
				}
			}

			fmt.Printf("%s %s\n", event.Status, event.Progress)
		}
	}

	return err
}

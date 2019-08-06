package main

import (
	"context"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/urfave/cli"
)

var app = cli.NewApp()

var sourceVolume, username, tag string

const targetVolume string = "/volume"

func flags() {
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "tag, t",
			Usage:       "Tag of managed volume.",
			Value:       "latest",
			Destination: &tag,
		},
		cli.StringFlag{
			Name:        "mount, m",
			Usage:       "Volume of container, where the backup will be performed.",
			Required:    true,
			Destination: &sourceVolume,
		},
		cli.StringFlag{
			Name:        "username, u",
			Usage:       "Username of account on Docker Hub.",
			Destination: &username,
		},
	}
}

func info() {
	app.Name = "Docker Volume Manager"
	app.Usage = "An CLI to control versions of Docker volumes using an Docker Registry."
	app.Author = "edmilson.santana"
	app.Version = "1.0.0"
}

func commands() {
	app.Commands = []cli.Command{
		{
			Name:    "backup",
			Aliases: []string{"b"},
			Usage:   "Docker container volume backup.",
			Action: func(c *cli.Context) error {
				log.Println("Starting backup process...")
				return backup()
			},
		},
		{
			Name:    "restore",
			Aliases: []string{"r"},
			Usage:   "Docker container volume restore.",
			Action: func(c *cli.Context) error {
				log.Println("Starting restore process...")
				return restore()
			},
		},
	}
}

func backup() error {

	cmd := []string{"tar", "-cvf", "/backup.tar", targetVolume}
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())

	var cont Container
	if err == nil {
		cont = Container{
			dockerClient: dockerClient,
			sourceVolume: sourceVolume,
			targetVolume: targetVolume,
			tag:          tag,
			username:     ""}

		err = handleFuncError(err, func() error {
			return cont.pull(ctx, "alpine")
		})
		err = handleFuncError(err, func() error {
			return cont.run(ctx, "alpine", cmd)
		})
		err = handleFuncError(err, func() error {
			return cont.commit(ctx)
		})
		err = handleFuncError(err, func() error {
			return cont.remove(ctx)
		})
	}

	return handleErrorMessage(err, "A imagem %s foi criada com sucesso.", cont.getImage(""))
}

func restore() error {
	cmd := []string{"tar", "-xvf", "/backup.tar"}
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())

	var cont Container
	if err == nil {
		cont = Container{
			dockerClient: dockerClient,
			sourceVolume: sourceVolume,
			targetVolume: targetVolume,
			tag:          tag,
			username:     username}

		err = handleFuncError(err, func() error {
			return cont.removeVolume(ctx)
		})
		err = handleFuncError(err, func() error {
			return cont.createVolume(ctx)
		})
		err = handleFuncError(err, func() error {
			return cont.pull(ctx, "")
		})
		err = handleFuncError(err, func() error {
			return cont.run(ctx, "", cmd)
		})
		err = handleFuncError(err, func() error {
			return cont.remove(ctx)
		})
	}

	return handleErrorMessage(err, "A imagem %s foi restaurada com sucesso", cont.getImage(""))
}

func main() {

	info()
	commands()
	flags()

	logError(app.Run(os.Args))
}

func handleFuncError(err error, fn func() error) error {
	if err == nil {
		err = fn()
	}
	return err
}

func handleErrorMessage(err error, msg, arg string) error {
	if err == nil {
		log.Printf(msg, arg)
	}
	return err
}

func logError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

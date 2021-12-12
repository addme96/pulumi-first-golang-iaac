package main

import (
	"fmt"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
	"path"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		stack := os.Getenv(pulumi.EnvStack)
		getwd, _ := os.Getwd()

		backendImageName := "backend"
		backend, err := docker.NewImage(ctx, "backend", &docker.ImageArgs{
			ImageName: pulumi.String(fmt.Sprintf("%v:%v", backendImageName, stack)),
			Build: docker.DockerBuildArgs{
				Context: pulumi.String(path.Join(getwd, "app", "backend")),
			},
			Registry: docker.ImageRegistryArgs{},
			SkipPush: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		frontendImageName := "frontend"
		frontend, err := docker.NewImage(ctx, "frontend", &docker.ImageArgs{
			ImageName: pulumi.String(fmt.Sprintf("%v:%v", frontendImageName, stack)),
			Build: docker.DockerBuildArgs{
				Context: pulumi.String(path.Join(getwd, "app", "frontend")),
			},
			Registry: docker.ImageRegistryArgs{},
			SkipPush: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		fmt.Print(backend.ImageName, frontend.ImageName)
		return nil
	})

}

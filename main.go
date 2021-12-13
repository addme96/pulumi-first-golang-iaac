package main

import (
	"fmt"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"os"
	"path"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		frontendPort := cfg.RequireInt("frontend_port")
		backendPort := cfg.RequireInt("backend_port")
		mongoPort := cfg.RequireInt("mongo_port")
		mongoHost := cfg.Require("mongo_host")
		mongoUsername := cfg.Require("mongo_username")
		mongoPassword := cfg.RequireSecret("mongo_password")
		database := cfg.Require("database")
		nodeEnvironment := cfg.Require("node_environment")

		ctx.Export("url", pulumi.Sprintf("http://localhost:%d", frontendPort))

		stack := ctx.Stack()
		getwd, _ := os.Getwd()

		backendImageName := "backend"
		backendImage, err := docker.NewImage(ctx, "backend", &docker.ImageArgs{
			ImageName: pulumi.Sprintf("%v:%v", backendImageName, stack),
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
		frontendImage, err := docker.NewImage(ctx, "frontend", &docker.ImageArgs{
			ImageName: pulumi.Sprintf("%v:%v", frontendImageName, stack),
			Build: docker.DockerBuildArgs{
				Context: pulumi.String(path.Join(getwd, "app", "frontend")),
			},
			Registry: docker.ImageRegistryArgs{},
			SkipPush: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		mongoImage, err := docker.NewRemoteImage(ctx, "mongo", &docker.RemoteImageArgs{
			Name: pulumi.String("mongo:bionic"),
		})
		if err != nil {
			return err
		}

		network := docker.Network{Name: pulumi.Sprintf("services-%s", stack)}

		mongoContainer, err := docker.NewContainer(ctx, "mongo_container", &docker.ContainerArgs{
			Image: mongoImage.Name,
			Name:  pulumi.Sprintf("mongo-%s", stack),
			Ports: docker.ContainerPortArray{docker.ContainerPortArgs{
				External: pulumi.Int(mongoPort),
				Internal: pulumi.Int(mongoPort),
			}},
			Envs: pulumi.StringArray{
				pulumi.Sprintf("MONGO_INITDB_ROOT_USERNAME=%s", mongoUsername),
				pulumi.Sprintf("MONGO_INITDB_ROOT_PASSWORD=%s", mongoPassword),
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				docker.ContainerNetworksAdvancedArgs{
					Aliases: pulumi.StringArray{pulumi.String("mongo")},
					Name:    network.Name,
				},
			},
		})
		if err != nil {
			return err
		}

		backendContainer, err := docker.NewContainer(ctx, "backend_container", &docker.ContainerArgs{
			Image: backendImage.BaseImageName,
			Name:  pulumi.Sprintf("backend-%s", stack),
			Ports: docker.ContainerPortArray{docker.ContainerPortArgs{
				Internal: pulumi.Int(backendPort),
				External: pulumi.Int(backendPort),
			}},
			Envs: pulumi.StringArray{
				pulumi.Sprintf("DATABASE_HOST=mongodb://%s:%s@%s:%s", mongoUsername, mongoPassword, mongoHost, mongoPort),
				pulumi.Sprintf("DATABASE_NAME=%s?authSource=admin", database),
				pulumi.Sprintf("NODE_ENV=%s", nodeEnvironment),
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				docker.ContainerNetworksAdvancedArgs{
					Name: network.Name,
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{mongoContainer}))
		if err != nil {
			return err
		}

		dataSeedContainer, err := docker.NewContainer(ctx, "data_seed", &docker.ContainerArgs{
			Image:   mongoImage.RepoDigest,
			MustRun: pulumi.Bool(false),
			Name:    pulumi.String("data_seed"),
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				docker.ContainerNetworksAdvancedArgs{
					Name: network.Name,
				},
			},
			Rm: pulumi.Bool(true),
			Mounts: docker.ContainerMountArray{docker.ContainerMountArgs{
				Source: pulumi.String(path.Join(getwd, "products.json")),
				Target: pulumi.String("/home/products.json"),
				Type:   pulumi.String("bind"),
			}},
			Command: pulumi.StringArray{
				pulumi.String("sh"),
				pulumi.String("-c"),
				pulumi.Sprintf("mongoimport --host mongo -u %s -p %s --authenticationDatabase admin --db cart --collection products --type json --file /home/products.json --jsonArray", mongoUsername, mongoPassword),
			},
		})
		if err != nil {
			return err
		}

		frontendContainer, err := docker.NewContainer(ctx, "frontend_container", &docker.ContainerArgs{
			Image: frontendImage.BaseImageName,
			Name:  pulumi.Sprintf("frontend-%s", stack),
			Ports: docker.ContainerPortArray{docker.ContainerPortArgs{
				Internal: pulumi.Int(frontendPort),
				External: pulumi.Int(frontendPort),
			}},
			Envs: pulumi.StringArray{
				pulumi.Sprintf("LISTEN_PORT=%d", frontendPort),
				pulumi.Sprintf("HTTP_PROXY=backend-%s:%d", stack, backendPort),
			},
			NetworksAdvanced: docker.ContainerNetworksAdvancedArray{
				docker.ContainerNetworksAdvancedArgs{
					Name: network.Name,
				},
			},
		})
		if err != nil {
			return err
		}
		fmt.Print(backendImage.ImageName, frontendImage.ImageName, mongoImage.Name, frontendPort, backendPort,
			mongoPort, network.Name, frontendContainer.Name, backendContainer.Name, dataSeedContainer.Name)

		return nil
	})

}

package main

import (
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

type mocks int

func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (mocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func TestInfrastructure(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		infra, err := createInfrastructure(ctx)
		assert.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)

		pulumi.All(infra.server.URN(), infra.server.Tags).ApplyT(func(all []interface{}) error {
			urn := all[0].(pulumi.URN)
			tags := all[1].(map[string]string)
			fmt.Println(tags)
			assert.Containsf(t, tags, "Name", "missing a Name tag on server %v", urn)
			wg.Done()
			return nil
		})
		pulumi.All(infra.server.URN(), infra.server.UserData).ApplyT(func(all []interface{}) error {
			urn := all[0].(pulumi.URN)
			userData := all[1].(string)
			assert.Empty(t, userData, "illegal use of userData on server %v", urn)
			return nil
		})
		pulumi.All(infra.server.URN(), infra.group.Ingress).ApplyT(func(all []interface{}) error {
			urn := all[0].(pulumi.URN)
			ingress := all[1].([]ec2.SecurityGroupIngress)
			for _, i := range ingress {
				openToInternet := false
				for _, b := range i.CidrBlocks {
					if b == "0.0.0.0/0" {
						openToInternet = true
						break
					}
				}
				assert.Falsef(t, i.FromPort == 22 && openToInternet, "illegal SHH port 22 open to the Internet (CIDR 0.0.0.0/0) on group %v", urn)
			}
			return nil
		})
		wg.Wait()
		return nil
	}, pulumi.WithMocks("project", "stack", mocks(0)))
	assert.NoError(t, err)
}

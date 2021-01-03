package wrapper

import (
	"github.com/aws/aws-sdk-go/service/ecs"
)

type Cluster struct {
	Name     string
	Arn      *string
	Running  int64
	Pending  int64
	Services []*Service
}

func (c *Cluster) ListServices(svc *ecs.ECS) (*ecs.ListServicesOutput, error) {
	result, err := svc.ListServices(&ecs.ListServicesInput{
		Cluster: c.Arn,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Cluster) DescribeServices(svc *ecs.ECS, services []*string) (*ecs.DescribeServicesOutput, error) {
	result, err := svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  c.Arn,
		Services: services,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

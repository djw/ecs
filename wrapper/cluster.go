package wrapper

import (
	"fmt"
	"os"
	"sync"

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

func getClusterList(svc *ecs.ECS) (*ecs.ListClustersOutput, error) {
	result, err := svc.ListClusters(&ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getClusterDescriptions(svc *ecs.ECS, clusters []*string) (*ecs.DescribeClustersOutput, error) {
	result, err := svc.DescribeClusters(&ecs.DescribeClustersInput{
		Clusters: clusters,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetClusters(svc *ecs.ECS) []Cluster {
	clustersList, err := getClusterList(svc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	clustersDescriptions, err := getClusterDescriptions(svc, clustersList.ClusterArns)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var clusters []Cluster
	for _, c := range clustersDescriptions.Clusters {
		cl := Cluster{
			Arn:     c.ClusterArn,
			Name:    *c.ClusterName,
			Running: *c.RunningTasksCount,
			Pending: *c.PendingTasksCount,
		}

		clusterServices, err := cl.ListServices(svc)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(clusterServices.ServiceArns) > 0 {
			clusterServiceDescriptions, _ := cl.DescribeServices(svc, clusterServices.ServiceArns)
			var wg sync.WaitGroup
			for _, s := range clusterServiceDescriptions.Services {
				ser := &Service{
					Cluster: cl,
					Name:    *s.ServiceName,
					Running: *s.RunningCount,
					Pending: *s.PendingCount,
				}

				wg.Add(1)
				go func(s *Service) {
					defer wg.Done()
					err := s.FetchTasks(svc)
					if err != nil {
						fmt.Println(err)
					}
				}(ser)

				cl.Services = append(cl.Services, ser)
			}
			wg.Wait()
		}
		clusters = append(clusters, cl)
	}
	return clusters
}

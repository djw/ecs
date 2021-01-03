package wrapper

import (
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/service/ecs"
)

// Cluster describes an ECS Cluster
type Cluster struct {
	Name     string
	Arn      *string
	Running  int64
	Pending  int64
	Services []*service
}

func (c *Cluster) listServices(svc *ecs.ECS) (*ecs.ListServicesOutput, error) {
	result, err := svc.ListServices(&ecs.ListServicesInput{
		Cluster: c.Arn,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Cluster) describeServices(svc *ecs.ECS, services []*string) (*ecs.DescribeServicesOutput, error) {
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

// GetClusters fetches a list of all clusters with descriptions
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

		clusterServices, err := cl.listServices(svc)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(clusterServices.ServiceArns) > 0 {
			clusterServiceDescriptions, _ := cl.describeServices(svc, clusterServices.ServiceArns)
			var wg sync.WaitGroup
			for _, s := range clusterServiceDescriptions.Services {
				ser := &service{
					Cluster: cl,
					Name:    *s.ServiceName,
					Running: *s.RunningCount,
					Pending: *s.PendingCount,
				}

				wg.Add(1)
				go func(s *service) {
					defer wg.Done()
					err := s.fetchTasks(svc)
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

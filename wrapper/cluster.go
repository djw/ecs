package wrapper

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// Cluster describes an ECS Cluster
type Cluster struct {
	Name     string
	Arn      *string
	Running  int32
	Pending  int32
	Services []*service
}

func (c *Cluster) listServices(client *ecs.Client) (*ecs.ListServicesOutput, error) {
	result, err := client.ListServices(context.TODO(), &ecs.ListServicesInput{
		Cluster: c.Arn,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Cluster) describeServices(client *ecs.Client, services []string) (*ecs.DescribeServicesOutput, error) {
	result, err := client.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
		Cluster:  c.Arn,
		Services: services,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getClusterList(client *ecs.Client) (*ecs.ListClustersOutput, error) {
	result, err := client.ListClusters(context.TODO(), &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getClusterDescriptions(client *ecs.Client, clusters []string) (*ecs.DescribeClustersOutput, error) {
	result, err := client.DescribeClusters(context.TODO(), &ecs.DescribeClustersInput{
		Clusters: clusters,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetClusters fetches a list of all clusters with descriptions.
// Results are returned to a channel.
func GetClusters(clusters chan<- *Cluster) {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	client := ecs.NewFromConfig(cfg)

	clustersList, err := getClusterList(client)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	clustersDescriptions, err := getClusterDescriptions(client, clustersList.ClusterArns)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var clusterWg sync.WaitGroup
	for _, c := range clustersDescriptions.Clusters {
		cl := &Cluster{
			Arn:     c.ClusterArn,
			Name:    *c.ClusterName,
			Running: c.RunningTasksCount,
			Pending: c.PendingTasksCount,
		}

		clusterWg.Add(1)
		go func(cl *Cluster) {
			defer clusterWg.Done()

			clusterServices, err := cl.listServices(client)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if len(clusterServices.ServiceArns) > 0 {
				clusterServiceDescriptions, _ := cl.describeServices(client, clusterServices.ServiceArns)
				var taskWg sync.WaitGroup
				for _, s := range clusterServiceDescriptions.Services {
					ser := &service{
						Cluster: *cl,
						Name:    *s.ServiceName,
						Running: s.RunningCount,
						Pending: s.PendingCount,
					}

					taskWg.Add(1)
					go func(s *service) {
						defer taskWg.Done()
						err := s.fetchTasks(client)
						if err != nil {
							fmt.Println(err)
						}
					}(ser)

					cl.Services = append(cl.Services, ser)
				}
				taskWg.Wait()
			}
			clusters <- cl
		}(cl)

	}
	go func() {
		clusterWg.Wait()
		close(clusters)
	}()
}

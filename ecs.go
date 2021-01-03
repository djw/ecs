package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/djw/ecs/wrapper"
	"github.com/olekukonko/tablewriter"
)

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

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ecs.New(sess)
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

	var clusters []wrapper.Cluster
	for _, c := range clustersDescriptions.Clusters {
		cl := wrapper.Cluster{
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
				ser := &wrapper.Service{
					Cluster: cl,
					Name:    *s.ServiceName,
					Running: *s.RunningCount,
					Pending: *s.PendingCount,
				}

				wg.Add(1)
				go func(s *wrapper.Service) {
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

	// Print as table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Running", "Pending"})
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})

	for _, c := range clusters {
		clusterRow := []string{
			c.Name,
			strconv.FormatInt(c.Running, 10),
			strconv.FormatInt(c.Pending, 10),
		}

		table.Rich(clusterRow, []tablewriter.Colors{
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
		})

		for _, s := range c.Services {
			row := []string{
				" - " + s.Name,
				strconv.FormatInt(s.Running, 10),
				strconv.FormatInt(s.Pending, 10),
			}
			table.Rich(row, []tablewriter.Colors{
				tablewriter.Colors{},
				tablewriter.Colors{},
				tablewriter.Colors{},
			})
			for _, t := range s.Tasks {
				task := fmt.Sprintf("  * %d (%v -> %v)",
					t.Revision,
					t.DesiredStatus,
					t.LastStatus,
				)
				row := []string{
					task,
					"",
					"",
				}
				table.Rich(row, []tablewriter.Colors{
					tablewriter.Colors{},
					tablewriter.Colors{},
					tablewriter.Colors{},
				})
			}
		}
	}

	table.Render()
}

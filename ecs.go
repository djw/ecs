package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
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

func listServices(svc *ecs.ECS, cluster *string) (*ecs.ListServicesOutput, error) {
	result, err := svc.ListServices(&ecs.ListServicesInput{
		Cluster: cluster,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func describeServices(svc *ecs.ECS, cluster *string, services []*string) (*ecs.DescribeServicesOutput, error) {
	result, err := svc.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  cluster,
		Services: services,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func listTasks(svc *ecs.ECS, cluster *string, service *string) (*ecs.DescribeTasksOutput, error) {
	taskList, err := svc.ListTasks(&ecs.ListTasksInput{
		Cluster:     cluster,
		ServiceName: service,
	})
	if err != nil {
		return nil, err
	}

	tasks, err := svc.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: cluster,
		Tasks:   taskList.TaskArns,
	})

	if err != nil {
		return nil, err
	}
	return tasks, nil
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

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Running", "Pending"})
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	for _, c := range clustersDescriptions.Clusters {
		clusterRow := []string{
			*c.ClusterName,
			strconv.FormatInt(*c.RunningTasksCount, 10),
			strconv.FormatInt(*c.PendingTasksCount, 10),
		}

		table.Rich(clusterRow, []tablewriter.Colors{
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
		})

		clusterServices, err := listServices(svc, c.ClusterArn)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(clusterServices.ServiceArns) > 0 {
			clusterServiceDescriptions, _ := describeServices(svc, c.ClusterArn, clusterServices.ServiceArns)
			for _, s := range clusterServiceDescriptions.Services {
				row := []string{
					" - " + *s.ServiceName,
					strconv.FormatInt(*s.RunningCount, 10),
					strconv.FormatInt(*s.PendingCount, 10),
				}
				table.Rich(row, []tablewriter.Colors{
					tablewriter.Colors{},
					tablewriter.Colors{},
					tablewriter.Colors{},
				})

				serviceTasks, err := listTasks(svc, c.ClusterArn, s.ServiceName)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				for _, t := range serviceTasks.Tasks {
					taskDef := strings.Split(*t.TaskDefinitionArn, ":")
					task := fmt.Sprintf("  * %v (%v -> %v)",
						taskDef[len(taskDef)-1],
						*t.DesiredStatus,
						*t.LastStatus,
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
	}

	table.Render()
}

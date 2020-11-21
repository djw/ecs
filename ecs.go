package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/olekukonko/tablewriter"
)

type cluster struct {
	name     string
	arn      *string
	running  int64
	pending  int64
	services []*service
}

type service struct {
	cluster cluster
	name    string
	running int64
	pending int64
	tasks   []task
}

type task struct {
	revision      int
	desiredStatus string
	lastStatus    string
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

func (s *service) fetchTasks(svc *ecs.ECS) error {
	taskList, err := svc.ListTasks(&ecs.ListTasksInput{
		Cluster:     s.cluster.arn,
		ServiceName: &s.name,
	})
	if err != nil {
		return err
	}

	taskDescriptions, err := svc.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: s.cluster.arn,
		Tasks:   taskList.TaskArns,
	})

	if err != nil {
		return err
	}

	var tasks []task
	for _, t := range taskDescriptions.Tasks {
		taskDef := strings.Split(*t.TaskDefinitionArn, ":")
		rev, _ := strconv.Atoi(taskDef[len(taskDef)-1])
		tk := task{
			revision:      rev,
			desiredStatus: *t.DesiredStatus,
			lastStatus:    *t.LastStatus,
		}
		tasks = append(tasks, tk)
	}
	s.tasks = tasks
	return nil
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

	var clusters []cluster
	for _, c := range clustersDescriptions.Clusters {
		cl := cluster{
			arn:     c.ClusterArn,
			name:    *c.ClusterName,
			running: *c.RunningTasksCount,
			pending: *c.PendingTasksCount,
		}

		clusterServices, err := listServices(svc, c.ClusterArn)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(clusterServices.ServiceArns) > 0 {
			clusterServiceDescriptions, _ := describeServices(svc, c.ClusterArn, clusterServices.ServiceArns)
			var wg sync.WaitGroup
			for _, s := range clusterServiceDescriptions.Services {
				ser := &service{
					cluster: cl,
					name:    *s.ServiceName,
					running: *s.RunningCount,
					pending: *s.PendingCount,
				}

				wg.Add(1)
				go func(s *service) {
					defer wg.Done()
					err := s.fetchTasks(svc)
					if err != nil {
						fmt.Println(err)
					}
				}(ser)

				cl.services = append(cl.services, ser)
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
			c.name,
			strconv.FormatInt(c.running, 10),
			strconv.FormatInt(c.pending, 10),
		}

		table.Rich(clusterRow, []tablewriter.Colors{
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
		})

		for _, s := range c.services {
			row := []string{
				" - " + s.name,
				strconv.FormatInt(s.running, 10),
				strconv.FormatInt(s.pending, 10),
			}
			table.Rich(row, []tablewriter.Colors{
				tablewriter.Colors{},
				tablewriter.Colors{},
				tablewriter.Colors{},
			})
			for _, t := range s.tasks {
				task := fmt.Sprintf("  * %d (%v -> %v)",
					t.revision,
					t.desiredStatus,
					t.lastStatus,
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

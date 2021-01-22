package wrapper

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type service struct {
	Cluster Cluster
	Name    string
	Running int32
	Pending int32
	Tasks   []task
}

func (s *service) fetchTasks(client *ecs.Client) error {
	taskList, err := client.ListTasks(context.TODO(), &ecs.ListTasksInput{
		Cluster:     s.Cluster.Arn,
		ServiceName: &s.Name,
	})
	if err != nil {
		return err
	}

	taskDescriptions, err := client.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
		Cluster: s.Cluster.Arn,
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
			Revision:      rev,
			DesiredStatus: *t.DesiredStatus,
			LastStatus:    *t.LastStatus,
		}
		tasks = append(tasks, tk)
	}
	s.Tasks = tasks
	return nil
}

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/djw/ecs/wrapper"
	"github.com/olekukonko/tablewriter"
)

func main() {
	clusters := wrapper.GetClusters()

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

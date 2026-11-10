package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// newMiscCmds: catalogue + capacity commands (every API endpoint reachable
// from the CLI — plan 7.1).
func newImagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "images",
		Short: "List the curated image catalogue",
		RunE: func(*cobra.Command, []string) error {
			c, err := newClient("")
			if err != nil {
				return err
			}
			var imgs []map[string]any
			if err := c.do("GET", "/images", nil, &imgs); err != nil {
				return err
			}
			return printOut(imgs, func() {
				rows := [][]string{}
				for _, i := range imgs {
					rows = append(rows, []string{
						fmt.Sprint(i["id"]), fmt.Sprint(i["name"]), fmt.Sprint(i["size_estimate"]), fmt.Sprint(i["description"])})
				}
				printTable([]string{"ID", "NAME", "SIZE", "DESCRIPTION"}, rows)
			})
		},
	}
}

func newTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "instance-types",
		Short: "List instance types",
		RunE: func(*cobra.Command, []string) error {
			c, err := newClient("")
			if err != nil {
				return err
			}
			var types []map[string]any
			if err := c.do("GET", "/instance-types", nil, &types); err != nil {
				return err
			}
			return printOut(types, func() {
				rows := [][]string{}
				for _, t := range types {
					rows = append(rows, []string{
						fmt.Sprint(t["id"]), fmt.Sprint(t["vcpu"]), fmt.Sprint(t["memory_mb"]), fmt.Sprint(t["description"])})
				}
				printTable([]string{"ID", "VCPU", "MEMORY MB", "DESCRIPTION"}, rows)
			})
		},
	}
}

func newCapacityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "capacity",
		Short: "Show host capacity, quotas, and the shared bridge network",
		RunE: func(*cobra.Command, []string) error {
			c, err := newClient("")
			if err != nil {
				return err
			}
			var b map[string]any
			if err := c.do("GET", "/capacity", nil, &b); err != nil {
				return err
			}
			return printOut(b, func() {
				fmt.Printf("host: %v CPUs, %v MB total; schedulable %v CPUs / %v MB (reserve applied)\n",
					b["total_cpus"], b["total_memory_mb"], b["schedulable_cpus"], b["schedulable_memory_mb"])
				fmt.Printf("used: %v instances (max %v), %v MB memory, %v MB volumes\n",
					b["instances"], b["max_instances"], b["used_memory_mb"], b["used_volume_mb"])
				fmt.Printf("quotas: %v MB memory, %v MB volumes\n", b["quota_memory_mb"], b["quota_volume_mb"])
				if net, ok := b["network"].(map[string]any); ok {
					fmt.Printf("network: %v (%v) — %v\n", net["name"], net["subnet"], net["note"])
				}
				_ = strings.TrimSpace // keep strings import stable
			})
		},
	}
}

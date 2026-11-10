package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// output flags
var outJSON bool

func printOut(v any, table func()) error {
	if outJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	table()
	return nil
}

// printTable renders aligned columns (no fancy borders).
func printTable(header []string, rows [][]string) {
	w := make([]int, len(header))
	for i, h := range header {
		w[i] = len(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(w) && len(c) > w[i] {
				w[i] = len(c)
			}
		}
	}
	for i, h := range header {
		fmt.Printf("%-*s  ", w[i], h)
	}
	fmt.Println()
	for _, r := range rows {
		for i := range header {
			fmt.Printf("%-*s  ", w[i], r[i])
		}
		fmt.Println()
	}
}

func argsToWidths(w []int) []any {
	out := make([]any, len(w))
	for i, x := range w {
		out[i] = x
	}
	return out
}

func instanceRow(in map[string]any) []string {
	ports := ""
	if ps, ok := in["ports"].([]any); ok {
		for _, p := range ps {
			pm := p.(map[string]any)
			ports += fmt.Sprintf("%v->%v ", pm["instance_port"], pm["host_port"])
		}
	}
	return []string{
		fmt.Sprint(in["id"]), fmt.Sprint(in["name"]), fmt.Sprint(in["state"]),
		fmt.Sprint(in["image_id"]), fmt.Sprint(in["type_id"]), fmt.Sprint(in["private_ip"]), ports,
	}
}

func newInstanceCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "instance", Short: "Manage Orbit instances"}
	cmd.PersistentFlags().BoolVar(&outJSON, "output", false, "output JSON instead of a table")
	cmd.PersistentFlags().Lookup("output").NoOptDefVal = "true"

	var image, typ, key, userDataFile string
	var ports []string
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an instance (async provisioning)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			body := map[string]any{"name": args[0], "image_id": image, "type_id": typ, "key_id": key}
			if userDataFile != "" {
				b, err := os.ReadFile(userDataFile)
				if err != nil {
					return err
				}
				body["user_data"] = string(b)
			}
			if len(ports) > 0 {
				ps := []map[string]any{}
				for _, p := range ports {
					// host:inst or host:inst:bind
					parts := strings.Split(p, ":")
					if len(parts) < 2 {
						return fmt.Errorf("port %q: want host:inst[:bind]", p)
					}
					pm := map[string]any{"instance_port": atoi(parts[1]), "host_port": atoi(parts[0])}
					if len(parts) > 2 {
						pm["bind"] = parts[2]
					}
					ps = append(ps, pm)
				}
				body["ports"] = ps
			}
			c, err := newClient("")
			if err != nil {
				return err
			}
			if err := acknowledgeTrustModel(); err != nil {
				return err
			}
			var in map[string]any
			if err := c.do("POST", "/instances", body, &in); err != nil {
				return err
			}
			return printInstance(in)
		},
	}
	create.Flags().StringVar(&image, "image", "", "image id (see orbit images)")
	create.Flags().StringVar(&typ, "type", "", "instance type (see orbit instance-types)")
	create.Flags().StringVar(&key, "key", "", "ssh key name (see orbit key list)")
	create.Flags().StringVar(&userDataFile, "user-data-file", "", "path to a startup script (#! required)")
	create.Flags().StringSliceVar(&ports, "port", nil, "publish ports host:inst[:loopback|all] (repeatable)")
	create.MarkFlagRequired("image")
	create.MarkFlagRequired("type")
	create.MarkFlagRequired("key")

	list := &cobra.Command{
		Use:   "list",
		Short: "List instances",
		RunE: func(_ *cobra.Command, _ []string) error {
			c, err := newClient("")
			if err != nil {
				return err
			}
			var ins []map[string]any
			if err := c.do("GET", "/instances", nil, &ins); err != nil {
				return err
			}
			return printOut(ins, func() {
				rows := [][]string{}
				for _, in := range ins {
					rows = append(rows, instanceRow(in))
				}
				printTable([]string{"ID", "NAME", "STATE", "IMAGE", "TYPE", "PRIVATE IP", "PORTS"}, rows)
			})
		},
	}

	get := &cobra.Command{
		Use:   "describe <id>",
		Short: "Describe an instance (with lifecycle events)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, err := newClient("")
			if err != nil {
				return err
			}
			var resp map[string]any
			if err := c.do("GET", "/instances/"+args[0], nil, &resp); err != nil {
				return err
			}
			return printOut(resp, func() {
				in := resp["instance"].(map[string]any)
				_ = printInstance(in)
				if evs, ok := resp["events"].([]any); ok {
					fmt.Println("\nEVENTS")
					rows := [][]string{}
					for _, e := range evs {
						em := e.(map[string]any)
						rows = append(rows, []string{fmt.Sprint(em["occurred_at"]), fmt.Sprint(em["type"]), fmt.Sprint(em["message"])})
					}
					printTable([]string{"TIME", "TYPE", "MESSAGE"}, rows)
				}
			})
		},
	}

	for _, a := range []string{"start", "stop", "reboot"} {
		act := a
		cmd.AddCommand(&cobra.Command{
			Use:   act + " <id>",
			Short: strings.ToUpper(act[:1]) + act[1:] + " an instance",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				c, err := newClient("")
				if err != nil {
					return err
				}
				var in map[string]any
				if err := c.do("POST", "/instances/"+args[0]+"/"+act, nil, &in); err != nil {
					return err
				}
				return printInstance(in)
			},
		})
	}

	var keepVol bool
	term := &cobra.Command{
		Use:   "terminate <id>",
		Short: "Terminate an instance (irreversible)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, err := newClient("")
			if err != nil {
				return err
			}
			var in map[string]any
			body := map[string]any{"keep_root_volume": keepVol}
			if err := c.do("POST", "/instances/"+args[0]+"/terminate", body, &in); err != nil {
				return err
			}
			return printInstance(in)
		},
	}
	term.Flags().BoolVar(&keepVol, "keep-root-volume", false, "keep the named root data volume")

	setPorts := &cobra.Command{
		Use:   "ports <id>",
		Short: "Replace published ports (stopped instances only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ps := []map[string]any{}
			for _, p := range ports {
				parts := strings.Split(p, ":")
				if len(parts) < 2 {
					return fmt.Errorf("port %q: want host:inst[:bind]", p)
				}
				pm := map[string]any{"instance_port": atoi(parts[1]), "host_port": atoi(parts[0])}
				if len(parts) > 2 {
					pm["bind"] = parts[2]
				}
				ps = append(ps, pm)
			}
			c, err := newClient("")
			if err != nil {
				return err
			}
			var in map[string]any
			if err := c.do("PUT", "/instances/"+args[0]+"/ports", map[string]any{"ports": ps}, &in); err != nil {
				return err
			}
			return printInstance(in)
		},
	}
	setPorts.Flags().StringSliceVar(&ports, "port", nil, "publish ports host:inst[:loopback|all] (repeatable; empty clears)")

	events := &cobra.Command{
		Use:   "events <id>",
		Short: "Instance lifecycle events",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, err := newClient("")
			if err != nil {
				return err
			}
			var evs []map[string]any
			if err := c.do("GET", "/instances/"+args[0]+"/events", nil, &evs); err != nil {
				return err
			}
			return printOut(evs, func() {
				rows := [][]string{}
				for _, e := range evs {
					rows = append(rows, []string{fmt.Sprint(e["occurred_at"]), fmt.Sprint(e["type"]), fmt.Sprint(e["message"])})
				}
				printTable([]string{"TIME", "TYPE", "MESSAGE"}, rows)
			})
		},
	}

	cmd.AddCommand(create, list, get, term, setPorts, events)
	return cmd
}

func printInstance(in map[string]any) error {
	return printOut(in, func() {
		printTable([]string{"ID", "NAME", "STATE", "IMAGE", "TYPE", "PRIVATE IP", "PORTS"}, [][]string{instanceRow(in)})
	})
}

func atoi(s string) int {
	n := 0
	fmt.Sscanf(s, "%d", &n)
	return n
}

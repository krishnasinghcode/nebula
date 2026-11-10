package main

import (
	"fmt"

	"orbit/internal/runtime"
	"orbit/internal/server"

	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the orbit control plane API (localhost only)",
		RunE: func(*cobra.Command, []string) error {
			token, err := server.LoadToken()
			if err != nil {
				return fmt.Errorf("load API token: %w", err)
			}
			fmt.Printf("orbit control plane listening on http://127.0.0.1:%d\n", port)
			dir, err := server.DataDir()
			if err != nil {
				return fmt.Errorf("resolve data dir: %w", err)
			}
			fmt.Printf("API token: %s\n", dir)
			// Best-effort runtime adapter for health reporting; the API keeps
			// serving (degraded) when Docker is down.
			var rt runtime.Adapter
			if da, err := runtime.NewDocker(); err == nil {
				rt = da
			}
			return server.Serve(port, server.New(token, rt))
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 7420, "port to listen on (loopback only)")
	return cmd
}

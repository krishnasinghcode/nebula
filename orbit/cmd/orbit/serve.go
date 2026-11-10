package main

import (
	"context"
	"fmt"
	"time"

	"orbit/internal/manager"
	"orbit/internal/runtime"
	"orbit/internal/server"
	"orbit/internal/store"

	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var port int
	var dataDir string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the orbit control plane API (localhost only)",
		RunE: func(*cobra.Command, []string) error {
			dir := dataDir
			if dir == "" {
				d, err := server.DataDir()
				if err != nil {
					return fmt.Errorf("resolve data dir: %w", err)
				}
				dir = d
			}
			token, err := server.LoadToken()
			if err != nil {
				return fmt.Errorf("load API token: %w", err)
			}
			st, err := store.Open(dir + "/orbit.db")
			if err != nil {
				return err
			}
			defer st.Close()

			rt, err := runtime.NewDocker()
			if err != nil {
				return fmt.Errorf("connect to docker: %w", err)
			}

			mgr := manager.New(st, rt)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mgr.StartReconciler(ctx, 5*time.Second)

			api := &server.API{Manager: mgr, Keys: st}
			fmt.Printf("orbit control plane listening on http://127.0.0.1:%d (data: %s)\n", port, dir)
			return server.Serve(port, server.New(token, rt, api))
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 7420, "port to listen on (loopback only)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "data directory (default ~/.orbit)")
	return cmd
}

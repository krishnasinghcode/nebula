package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"orbit/internal/server"

	"github.com/spf13/cobra"
)

func newSSHCmd() *cobra.Command {
	var url string
	cmd := &cobra.Command{
		Use:   "ssh <instance>",
		Short: "SSH into a running instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c, err := newClient(url)
			if err != nil {
				return err
			}
			var resp struct {
				Instance struct {
					ID    string `json:"id"`
					State string `json:"state"`
					KeyID string `json:"key_id"`
					Ports []struct {
						InstancePort int `json:"instance_port"`
						HostPort     int `json:"host_port"`
					} `json:"ports"`
				} `json:"instance"`
			}
			if err := c.do("GET", "/instances/"+args[0], nil, &resp); err != nil {
				return err
			}
			in := resp.Instance
			if in.State != "running" {
				return fmt.Errorf("instance %s is %s; SSH needs a running instance", args[0], in.State)
			}
			hostPort := 0
			for _, p := range in.Ports {
				if p.InstancePort == 22 {
					hostPort = p.HostPort
				}
			}
			if hostPort == 0 {
				return fmt.Errorf("instance %s has no published SSH port", args[0])
			}
			dir, err := server.DataDir()
			if err != nil {
				return err
			}
			keyPath := fmt.Sprintf("%s/keys/%s", dir, in.KeyID)
			if _, err := os.Stat(keyPath); err != nil {
				return fmt.Errorf("private key for %q not found at %s (create it with `orbit key create %s`)",
					in.KeyID, keyPath, in.KeyID)
			}
			sshBin, err := exec.LookPath("ssh")
			if err != nil {
				return fmt.Errorf("ssh client not found on PATH: %w", err)
			}
			// Pass through argv; stdin/stdout inherited so interactive sessions work.
			cmdArgs := []string{
				"ssh", "-i", keyPath,
				"-o", "StrictHostKeyChecking=no", // shortcut: local-only, ephemeral instances; pin known_hosts when instances get stable addresses
				"-o", "UserKnownHostsFile=/dev/null",
				"-p", fmt.Sprint(hostPort),
				"orbit@127.0.0.1",
			}
			if len(os.Args) > 2 {
				cmdArgs = append(cmdArgs, os.Args[2:]...)
			}
			ps, err := os.FindProcess(os.Getpid())
			_ = ps
			_ = err
			return syscallExec(sshBin, cmdArgs)
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "control plane URL (default http://127.0.0.1:7420)")
	return cmd
}

// syscallExec replaces the process (Unix) or runs and waits (Windows).
func syscallExec(bin string, args []string) error {
	if isWindows() {
		cmd := exec.Command(bin, args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	}
	return syscall.Exec(bin, args, os.Environ())
}

func isWindows() bool { return os.PathSeparator == '\\' }

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"orbit/internal/server"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func newKeyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "key", Short: "Manage SSH keys"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "create <name>",
			Short: "Generate an Ed25519 keypair and register the public key",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				name := args[0]
				dir, err := server.DataDir()
				if err != nil {
					return err
				}
				keysDir := filepath.Join(dir, "keys")
				if err := os.MkdirAll(keysDir, 0o700); err != nil {
					return err
				}
				privPath := filepath.Join(keysDir, name)
				if _, err := os.Stat(privPath); err == nil {
					return fmt.Errorf("key %q already exists locally at %s", name, privPath)
				}
				pub, priv, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					return err
				}
				sshPub, err := ssh.NewPublicKey(pub)
				if err != nil {
					return err
				}
				pubLine := string(ssh.MarshalAuthorizedKey(sshPub))

				// Private key in OpenSSH PEM format for the ssh client.
				privPEM, err := ssh.MarshalPrivateKey(priv, "orbit:"+name)
				if err != nil {
					return err
				}
				if err := os.WriteFile(privPath, pem.EncodeToMemory(privPEM), 0o600); err != nil {
					return err
				}

				c, err := newClient("")
				if err != nil {
					return err
				}
				key := map[string]any{"name": name, "public_key": pubLine}
				return c.do("POST", "/keys", key, nil)
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List registered SSH keys",
			RunE: func(*cobra.Command, []string) error {
				c, err := newClient("")
				if err != nil {
					return err
				}
				var keys []map[string]any
				if err := c.do("GET", "/keys", nil, &keys); err != nil {
					return err
				}
				for _, k := range keys {
					fmt.Printf("%s\t%s\n", k["name"], k["fingerprint"])
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "rm <name>",
			Short: "Delete a registered SSH key",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				c, err := newClient("")
				if err != nil {
					return err
				}
				return c.do("DELETE", "/keys/"+args[0], nil, nil)
			},
		},
	)
	return cmd
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orbit/internal/server"
)

// trustStatement is security.md §1 — shown once before the first instance
// create (NFR-4 / security gate 7).
const trustStatement = `
IMPORTANT — trust model (docs/features/orbit/design/security.md §1):

Orbit with the container runtime is NOT a secure multi-tenant cloud
hypervisor. Containers share the host kernel; a container root with
sufficient privileges, a kernel vulnerability, or a misconfigured mount can
lead to host compromise.

  - Your own workloads (dev tools, apps you wrote): trusted enough.
  - Untrusted third-party software: NOT safe here; use a real VM.
  - Deliberate attackers with shell access: out of threat model.

Defences raise the bar; they do not close it.
`

// acknowledgeTrustModel shows the statement once per install and records the
// acknowledgement in the data dir.
func acknowledgeTrustModel() error {
	dir, err := server.DataDir()
	if err != nil {
		return err
	}
	ackPath := filepath.Join(dir, "trust-acknowledged")
	if _, err := os.Stat(ackPath); err == nil {
		return nil
	}
	fmt.Print(trustStatement)
	fmt.Print("I understand Orbit instances are containers sharing my host kernel. Continue? [y/N] ")
	rd := bufio.NewReader(os.Stdin)
	line, _ := rd.ReadString('\n')
	if !strings.EqualFold(strings.TrimSpace(line), "y") && !strings.EqualFold(strings.TrimSpace(line), "yes") {
		return fmt.Errorf("instance creation cancelled (run `orbit instance create` again to accept)")
	}
	return os.WriteFile(ackPath, []byte("acknowledged\n"), 0o600)
}

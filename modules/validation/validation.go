package validation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kaminskip88/terraform-test/modules/basic"

	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
)

// Pinned so a floating chef/inspec:latest tag cannot change exit-code behavior.
const inspecImage = "chef/inspec:5.24.24"

const sshConfig = `Host *
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
`

func RunInspecDocker(t *testing.T, s basic.Scenario, spec string, user string, sudo bool) {
	target := terraform.OutputRequired(t, s.TFOpts, "ip_address")
	sshKey := terraform.OutputRequired(t, s.TFOpts, "ssh_key_priv")

	require.NoError(t, waitForTCP(net.JoinHostPort(target, "22"), 2*time.Minute))

	sshKeyPath := filepath.Join(s.ScenarioPath, "ssh_priv")
	sshConfigPath := filepath.Join(s.ScenarioPath, "ssh_config")
	require.NoError(t, os.WriteFile(sshKeyPath, []byte(sshKey), 0o600))
	defer func() { require.NoError(t, os.Remove(sshKeyPath)) }()
	require.NoError(t, os.WriteFile(sshConfigPath, []byte(sshConfig), 0o644))
	defer func() { require.NoError(t, os.Remove(sshConfigPath)) }()

	specPath := filepath.Join(s.ModulePath, spec)
	opts := inspecDockerOptions(specPath, sshKeyPath, sshConfigPath, inspecCommand(user, target, sudo))

	docker.Run(t, inspecImage, &opts)
}

func inspecCommand(user string, target string, sudo bool) []string {
	cmd := []string{
		"exec", "/spec", "--no-create-lockfile", "--no-distinct-exit",
		"--target", fmt.Sprintf("ssh://%s@%s", user, target),
		"--key-files", "/ssh/ssh_priv",
		"--ssh-config-file", "/ssh/config",
	}
	if sudo {
		cmd = append(cmd, "--sudo")
	}
	return cmd
}

func inspecDockerOptions(specPath string, sshKeyPath string, sshConfigPath string, command []string) docker.RunOptions {
	return docker.RunOptions{
		Remove:  true,
		Command: command,
		Volumes: []string{
			specPath + ":/spec",
			sshKeyPath + ":/ssh/ssh_priv",
			sshConfigPath + ":/ssh/config",
		},
		EnvironmentVariables: []string{
			"CHEF_LICENSE=accept",
		},
	}
}

func waitForTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timed out waiting for %s: %w", addr, lastErr)
		}
		conn, err := net.DialTimeout("tcp", addr, remaining)
		if err == nil {
			return conn.Close()
		}
		lastErr = err
		time.Sleep(min(time.Second, time.Until(deadline)))
	}
}

func AssertIPv4(t *testing.T, s string) {
	ip := net.ParseIP(s)
	require.NotNil(t, ip, "Not match IPv4 pattern")
	require.NotNil(t, ip.To4(), "Not match IPv4 pattern")
}

package validation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaminskip88/terraform-test/modules/basic"

	"github.com/gruntwork-io/terratest/modules/docker"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
)

func RunInspecDocker(t *testing.T, s basic.Scenario, spec string, user string, sudo bool) {
	target := terraform.OutputRequired(t, s.TFOpts, "ipv4_address_public")
	sshKey := terraform.OutputRequired(t, s.TFOpts, "ssh_key_priv")

	sshKeyPath := filepath.Join(s.ScenarioPath, "ssh_priv")
	err := os.WriteFile(sshKeyPath, []byte(sshKey), 0o600)
	require.NoError(t, err)

	specPath := filepath.Join(s.ModulePath, spec)

	dCommand := []string{
		"exec", "/spec", "--no-create-lockfile",
		"--target", fmt.Sprintf("ssh://%s@%s", user, target),
		"--key-files", "/ssh/ssh_priv",
	}

	if sudo {
		dCommand = append(dCommand, "--sudo")
	}

	dOpts := docker.RunOptions{
		Command: dCommand,
		Volumes: []string{
			specPath + ":" + "/spec",
			sshKeyPath + ":" + "/ssh/ssh_priv",
		},
		EnvironmentVariables: []string{
			"CHEF_LICENSE=accept",
		},
	}

	docker.Run(t, "chef/inspec", &dOpts)

	err = os.Remove(sshKeyPath)
	require.NoError(t, err)
}

func AssertIPv4(t *testing.T, s string) {
	ip := net.ParseIP(s)
	require.NotNil(t, ip, "Not match IPv4 pattern")
	require.NotNil(t, ip.To4(), "Not match IPv4 pattern")
}

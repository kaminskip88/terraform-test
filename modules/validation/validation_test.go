package validation

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInspecImageIsPinned(t *testing.T) {
	require.Contains(t, inspecImage, ":")
	require.NotContains(t, inspecImage, ":latest")
}

func TestInspecCommandIncludesSSHConfig(t *testing.T) {
	cmd := inspecCommand("provision", "10.0.0.1", true)
	require.Contains(t, cmd, "--ssh-config-file")
	require.Contains(t, cmd, "/ssh/config")
	require.Contains(t, cmd, "--no-distinct-exit")
	require.Contains(t, cmd, "--sudo")
}

func TestInspecDockerOptions(t *testing.T) {
	opts := inspecDockerOptions("/spec", "/tmp/key", "/tmp/config", []string{"exec", "/spec"})
	require.True(t, opts.Remove)
	require.Contains(t, opts.Volumes, "/spec:/spec")
	require.Contains(t, opts.Volumes, "/tmp/key:/ssh/ssh_priv")
	require.Contains(t, opts.Volumes, "/tmp/config:/ssh/config")
}

func TestWaitForTCP_dialTimeoutBoundedByDeadline(t *testing.T) {
	// 10.255.255.1 is a non-routable address, so SYNs are dropped rather than refused.
	start := time.Now()
	err := waitForTCP("10.255.255.1:22", 300*time.Millisecond)
	require.Error(t, err)
	require.Less(t, time.Since(start), 2*time.Second)
}

func TestWaitForTCP_succeedsWhenPortOpen(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	require.NoError(t, waitForTCP(ln.Addr().String(), 2*time.Second))
}

func TestWaitForTCP_timesOutWhenClosed(t *testing.T) {
	err := waitForTCP("127.0.0.1:1", 200*time.Millisecond)
	require.Error(t, err)
}

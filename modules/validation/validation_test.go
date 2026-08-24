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
	require.Contains(t, cmd, "--sudo")
}

func TestInspecDockerOptionsRemoveContainer(t *testing.T) {
	opts := inspecDockerOptions("/spec", "/tmp/key", []string{"exec", "/spec"})
	require.True(t, opts.Remove)
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

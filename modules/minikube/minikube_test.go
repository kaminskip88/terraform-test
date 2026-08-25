package minikube

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig_nodesIsOne(t *testing.T) {
	cfg := DefaultConfig()
	require.Equal(t, 1, cfg.Nodes)
	require.Empty(t, cfg.Name)
	require.Empty(t, cfg.CPUs)
	require.Empty(t, cfg.Memory)
	require.Empty(t, cfg.Addons)
}

func TestStartArgs_nameOnlyUsesOneNode(t *testing.T) {
	got := startArgs(Config{Name: "tf-vault"})
	require.Equal(t, []string{"start", "--profile", "tf-vault", "--nodes", "1"}, got)
}

func TestStartArgs_zeroNodesBecomesOne(t *testing.T) {
	got := startArgs(Config{Name: "tf-vault", Nodes: 0})
	require.Equal(t, []string{"start", "--profile", "tf-vault", "--nodes", "1"}, got)
}

func TestStartArgs_threeNodes(t *testing.T) {
	got := startArgs(Config{Name: "tf-vault", Nodes: 3})
	require.Equal(t, []string{"start", "--profile", "tf-vault", "--nodes", "3"}, got)
}

func TestStartArgs_cpusAndMemory(t *testing.T) {
	got := startArgs(Config{Name: "tf-vault", Nodes: 1, CPUs: "2", Memory: "4g"})
	require.Equal(t, []string{
		"start", "--profile", "tf-vault", "--nodes", "1",
		"--cpus", "2", "--memory", "4g",
	}, got)
}

func TestStartArgs_omitsEmptyCpusAndMemory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Name = "tf-vault"
	got := startArgs(cfg)
	require.Equal(t, []string{"start", "--profile", "tf-vault", "--nodes", "1"}, got)
}

func TestStartArgs_addonsTrimsWhitespaceAndSkipsBlank(t *testing.T) {
	got := startArgs(Config{
		Name:   "tf-vault",
		Nodes:  1,
		Addons: []string{" ", " ingress "},
	})
	require.Equal(t, []string{
		"start", "--profile", "tf-vault", "--nodes", "1",
		"--addons", "ingress",
	}, got)
}

func TestStartArgs_cpusWithoutMemory(t *testing.T) {
	got := startArgs(Config{Name: "tf-vault", Nodes: 1, CPUs: "2"})
	require.Equal(t, []string{
		"start", "--profile", "tf-vault", "--nodes", "1",
		"--cpus", "2",
	}, got)
}

func TestStartArgs_repeatedAddonsSkipsBlank(t *testing.T) {
	got := startArgs(Config{
		Name:   "tf-vault",
		Nodes:  1,
		Addons: []string{"ingress", "", "metrics-server"},
	})
	require.Equal(t, []string{
		"start", "--profile", "tf-vault", "--nodes", "1",
		"--addons", "ingress", "--addons", "metrics-server",
	}, got)
}

func TestDeleteArgs_profileOnly(t *testing.T) {
	got := deleteArgs(Config{Name: "tf-vault", CPUs: "2", Memory: "4g", Nodes: 3, Addons: []string{"ingress"}})
	require.Equal(t, []string{"delete", "--profile", "tf-vault"}, got)
}

func TestStart_hookName(t *testing.T) {
	hook := Start(DefaultConfig())
	require.Equal(t, "minikube-start", hook.Name)
	require.NotNil(t, hook.Func)
}

func TestDelete_hookName(t *testing.T) {
	hook := Delete(DefaultConfig())
	require.Equal(t, "minikube-delete", hook.Name)
	require.NotNil(t, hook.Func)
}

package minikube

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kaminskip88/terraform-test/modules/basic"

	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/stretchr/testify/require"
)

type Config struct {
	Name   string
	CPUs   string
	Memory string
	Nodes  int
	Addons []string
}

func DefaultConfig() Config {
	return Config{Nodes: 1}
}

func Start(cfg Config) basic.Prepare {
	return basic.Prepare{
		Name: "minikube-start",
		Func: func(t *testing.T, _ basic.Scenario) {
			require.NotEmpty(t, cfg.Name, "minikube Config.Name is required")
			shell.RunCommand(t, shell.Command{
				Command: "minikube",
				Args:    startArgs(cfg),
			})
		},
	}
}

func Delete(cfg Config) basic.Teardown {
	return basic.Teardown{
		Name: "minikube-delete",
		Func: func(t *testing.T, _ basic.Scenario) {
			require.NotEmpty(t, cfg.Name, "minikube Config.Name is required")
			shell.RunCommand(t, shell.Command{
				Command: "minikube",
				Args:    deleteArgs(cfg),
			})
		},
	}
}

func startArgs(cfg Config) []string {
	nodes := cfg.Nodes
	if nodes == 0 {
		nodes = 1
	}
	args := []string{"start", "--profile", cfg.Name, "--nodes", strconv.Itoa(nodes)}
	if cfg.CPUs != "" {
		args = append(args, "--cpus", cfg.CPUs)
	}
	if cfg.Memory != "" {
		args = append(args, "--memory", cfg.Memory)
	}
	for _, addon := range cfg.Addons {
		if strings.TrimSpace(addon) == "" {
			continue
		}
		args = append(args, "--addons", strings.TrimSpace(addon))
	}
	return args
}

func deleteArgs(cfg Config) []string {
	return []string{"delete", "--profile", cfg.Name}
}

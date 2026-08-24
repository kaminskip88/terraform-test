package basic

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathContainsIgnoredPath_matchesWholePathSegment(t *testing.T) {
	conf := &Conf{IgnoredPaths: []string{"tests", "images"}}

	require.True(t, PathContainsIgnoredPath("/module/tests/inspec/x.rb", conf))
	require.True(t, PathContainsIgnoredPath("/module/images/packer/x.json", conf))
	require.True(t, PathContainsIgnoredPath("tests/inspec/x.rb", conf))
}

func TestPathContainsIgnoredPath_doesNotMatchSubstringSegments(t *testing.T) {
	conf := &Conf{IgnoredPaths: []string{"tests", "images"}}

	require.False(t, PathContainsIgnoredPath("/module/my-tests/x.go", conf))
	require.False(t, PathContainsIgnoredPath("/module/my-images/disk.img", conf))
	require.False(t, PathContainsIgnoredPath("/module/examples/gce/main.tf", conf))
}

func TestSelectScenarios_unknownNameFails(t *testing.T) {
	_, err := selectScenarios([]Scenario{{Name: "gce"}}, "vsphere")
	require.Error(t, err)
	require.Contains(t, err.Error(), "vsphere")
}

func TestSelectScenarios_allWhenUnsetOrAll(t *testing.T) {
	in := []Scenario{{Name: "gce"}, {Name: "vsphere"}}

	got, err := selectScenarios(in, "all")
	require.NoError(t, err)
	require.Equal(t, in, got)

	got, err = selectScenarios(in, "")
	require.NoError(t, err)
	require.Equal(t, in, got)
}

func TestSelectScenarios_oneMatch(t *testing.T) {
	in := []Scenario{{Name: "gce"}, {Name: "vsphere"}}
	got, err := selectScenarios(in, "gce")
	require.NoError(t, err)
	require.Equal(t, []Scenario{{Name: "gce"}}, got)
}

func TestValidateStage_unknownFails(t *testing.T) {
	require.Error(t, validateStage("typo"))
}

func TestValidateStage_knownOrEmptyOK(t *testing.T) {
	for _, s := range []string{"", "apply", "validate", "destroy", "build_scenario", "ssh"} {
		require.NoError(t, validateStage(s), s)
	}
}

func TestSSHRequested_stageSSHWithoutScenarioEnv(t *testing.T) {
	require.True(t, sshRequested("ssh", "", "gce"))
}

func TestSSHRequested_onlyWhenStageIsSSH(t *testing.T) {
	require.False(t, sshRequested("", "", "gce"))
	require.False(t, sshRequested("apply", "gce", "gce"))
	require.True(t, sshRequested("ssh", "gce", "gce"))
}

func TestUtilFilter_sshStageWithoutScenarioEnv(t *testing.T) {
	t.Setenv("TF_TEST_STAGE", "ssh")
	os.Unsetenv("TF_TEST_SCENARIO")
	require.True(t, utilFilter("ssh"))
}

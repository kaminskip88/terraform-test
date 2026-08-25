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
	for _, s := range []string{"", "apply", "validate", "destroy", "build_scenario", "ssh", "prepare", "teardown"} {
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

func TestCollectPrepareHooks_sharedThenScenario(t *testing.T) {
	shared := []Prepare{{Name: "s1"}, {Name: "s2"}}
	scenario := Scenario{Prepare: []Prepare{{Name: "p1"}, {Name: "p2"}}}
	got := collectPrepareHooks(shared, scenario)
	require.Equal(t, []string{"s1", "s2", "p1", "p2"}, prepareNames(got))
}

func TestCollectPrepareHooks_empty(t *testing.T) {
	require.Empty(t, collectPrepareHooks(nil, Scenario{}))
	require.Empty(t, collectPrepareHooks([]Prepare{}, Scenario{}))
}

func TestCollectTeardownHooks_reverseStack(t *testing.T) {
	shared := []Teardown{{Name: "s1"}, {Name: "s2"}}
	scenario := Scenario{Teardown: []Teardown{{Name: "p1"}, {Name: "p2"}}}
	got := collectTeardownHooks(shared, scenario)
	require.Equal(t, []string{"p2", "p1", "s2", "s1"}, teardownNames(got))
}

func TestCollectTeardownHooks_empty(t *testing.T) {
	require.Empty(t, collectTeardownHooks(nil, Scenario{}))
	require.Empty(t, collectTeardownHooks([]Teardown{}, Scenario{}))
}

func prepareNames(in []Prepare) []string {
	out := make([]string, len(in))
	for i, p := range in {
		out[i] = p.Name
	}
	return out
}

func teardownNames(in []Teardown) []string {
	out := make([]string, len(in))
	for i, p := range in {
		out[i] = p.Name
	}
	return out
}

func TestShouldRemoveTempDir(t *testing.T) {
	tests := []struct {
		name               string
		stage              string
		after              string
		teardownHookCount  int
		want               bool
	}{
		{name: "full cycle with teardown hooks after teardown", stage: "", after: "teardown", teardownHookCount: 1, want: true},
		{name: "full cycle with teardown hooks after destroy", stage: "", after: "destroy", teardownHookCount: 1, want: false},
		{name: "full cycle without teardown hooks after destroy", stage: "", after: "destroy", teardownHookCount: 0, want: true},
		{name: "full cycle without teardown hooks after teardown", stage: "", after: "teardown", teardownHookCount: 0, want: false},
		{name: "destroy stage after destroy with hooks", stage: "destroy", after: "destroy", teardownHookCount: 2, want: true},
		{name: "destroy stage after teardown", stage: "destroy", after: "teardown", teardownHookCount: 2, want: false},
		{name: "teardown stage does not remove", stage: "teardown", after: "teardown", teardownHookCount: 1, want: false},
		{name: "prepare stage does not remove", stage: "prepare", after: "destroy", teardownHookCount: 0, want: false},
		{name: "apply stage does not remove", stage: "apply", after: "destroy", teardownHookCount: 0, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRemoveTempDir(tc.stage, tc.after, tc.teardownHookCount)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestFilter_skipsPrepareWhenStageIsApply(t *testing.T) {
	t.Setenv("TF_TEST_STAGE", "apply")
	t.Run("Prepare", func(t *testing.T) {
		filter(t, "prepare")
		t.Fatal("filter should have skipped prepare when TF_TEST_STAGE=apply")
	})
}

func TestFilter_runsPrepareWhenStageIsPrepare(t *testing.T) {
	t.Setenv("TF_TEST_STAGE", "prepare")
	t.Run("Prepare", func(t *testing.T) {
		filter(t, "prepare")
	})
}

func TestFilter_skipsTeardownWhenStageIsDestroy(t *testing.T) {
	t.Setenv("TF_TEST_STAGE", "destroy")
	t.Run("Teardown", func(t *testing.T) {
		filter(t, "teardown")
		t.Fatal("filter should have skipped teardown when TF_TEST_STAGE=destroy")
	})
}

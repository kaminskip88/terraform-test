package basic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/files"
	terratest_logger "github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	terratest_testing "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Conf struct {
	TmpDir string
	RunDir string
	IgnoredPaths []string
}

func DefaultConf() *Conf {
	return &Conf{
		TmpDir: ".terratest",
		RunDir: "examples",
		IgnoredPaths: []string{
			"images",
			"tests",
		},
	}
}

type Scenario struct {
	Name         string
	Source       string
	ModulePath   string
	TempPath     string
	ScenarioPath string
	TFOpts       *terraform.Options
}

type Validation struct {
	Name string
	Func func(*testing.T, Scenario)
}

type zerologTestLogger struct {
	log zerolog.Logger
}

func (l *zerologTestLogger) Log(t terratest_testing.TestingT, args ...interface{}) {
	l.log.Debug().Msgf("%+v", args...)
}

func (l *zerologTestLogger) Logf(t terratest_testing.TestingT, format string, args ...interface{}) {
	l.log.Debug().Msgf(format, args...)
}

func Run(t *testing.T, conf *Conf, scenarios []Scenario, vals []Validation) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	terratest_logger.Default = terratest_logger.New(&zerologTestLogger{
		log: zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			Level(zerolog.DebugLevel).
			With().
			Timestamp().
			Logger(),
	})

	if len(scenarios) == 0 {
		scenarios = append(scenarios, Scenario{
			Name: "main",
		})
	}

	if err := validateStage(os.Getenv("TF_TEST_STAGE")); err != nil {
		require.NoError(t, err)
	}

	runScenario, ok := os.LookupEnv("TF_TEST_SCENARIO")

	if ok {
		log.Info().Msgf("Scenario - %s", runScenario)
	} else {
		log.Info().Msg("Scenario not set, running all")
		runScenario = "all"
	}

	selected, err := selectScenarios(scenarios, runScenario)
	require.NoError(t, err)

	for _, s := range selected {
		name := cases.Title(language.English).String(s.Name)
		t.Run(name, func(t *testing.T) {
			scenarioTest(t, conf, s, vals)
		})
	}
}

func scenarioTest(t *testing.T, conf *Conf, scenario Scenario, vals []Validation) {
	log := log.With().Str("scenario", scenario.Name).Logger()
	// t.Parallel()

	// get paths
	testDir, err := os.Getwd()
	if err != nil {
		require.NoError(t, err)
	}
	modulePath := filepath.Dir(testDir)
	tempPath := filepath.Join(modulePath, conf.TmpDir)

	var scenarioSrc string
	if scenario.Source == "" {
		scenarioSrc = scenario.Name
	} else {
		scenarioSrc = scenario.Source
	}

	scenarioPath := filepath.Join(tempPath, scenarioSrc)

	// set attributes to pass to validations
	scenario.ModulePath = modulePath
	scenario.TempPath = tempPath
	scenario.ScenarioPath = scenarioPath

	t.Run("BuildScenario", func(t *testing.T) {
		// Create sceario dir if necessary
		d, err := os.Stat(scenarioPath)
		if err == nil && d.IsDir() {
			log.Info().Msg("using existing scenario folder")
		} else {
			log.Info().Msg("creating scenario folder")
			// make temp dir
			if err := os.MkdirAll(scenarioPath, 0755); err != nil {
				require.NoError(t, err)
			}
		}

		// Populate scenario folder if necessary
		var copyRequired bool
		v, ok := os.LookupEnv("TF_TEST_STAGE")
		if ok && v == "build_scenario" {
			copyRequired = true
			log.Info().Msg("TF_TEST_STAGE=build_scenario building scenario dir")
		} else {
			f, err := os.ReadDir(scenarioPath)
			require.NoError(t, err)
			if len(f) > 0 {
				log.Info().Msg("scenario folder not empty, skip scenario dir build")
			} else {
				copyRequired = true
			}
		}

		if copyRequired {
			// filter files
			fileFilter := func(path string) bool {
				return !files.PathContainsHiddenFileOrFolder(path) &&
					!files.PathContainsTerraformStateOrVars(path) &&
					!PathContainsIgnoredPath(path, conf)
			}

			log.Info().Msg("populate scenario folder")
			// copy files to temp dir
			err = files.CopyFolderContentsWithFilter(modulePath, scenarioPath, fileFilter)
			if err != nil {
				require.NoError(t, err)
			}
		}
	})

	scenarioRunDir := filepath.Join(scenarioPath, conf.RunDir)

	d, err := os.Stat(filepath.Join(scenarioRunDir, scenario.Name))
	if err == nil && d.IsDir() {
		scenarioRunDir = filepath.Join(scenarioRunDir, scenario.Name)
	}

	log.Info().Msgf("Run TF in %s", scenarioRunDir)

	tfOpts := &terraform.Options{
		TerraformBinary: "tofu",
		TerraformDir:    scenarioRunDir,
		// Vars:         tfVars,
	}

	scenario.TFOpts = tfOpts

	// Defer destroy early in case apply failed
	defer t.Run("Destroy", func(t *testing.T) {
		filter(t, "destroy")
		terraform.Destroy(t, tfOpts)

		// remove temp folder
		err := os.RemoveAll(scenarioPath)
		require.NoError(t, err)
	})

	t.Run("Apply", func(t *testing.T) {
		filter(t, "apply")
		terraform.InitAndApply(t, tfOpts)
	})

	t.Run("Validate", func(t *testing.T) {
		filter(t, "validate")

		for _, v := range vals {
			v := v
			t.Run(v.Name, func(t *testing.T) {
				v.Func(t, scenario)
			})
		}
	})

	// Utility steps

	if sshRequested(os.Getenv("TF_TEST_STAGE"), os.Getenv("TF_TEST_SCENARIO"), scenario.Name) {
		t.Run("ssh", func(t *testing.T) {
			target := terraform.OutputRequired(t, scenario.TFOpts, "ip_address")
			sshKey := terraform.OutputRequired(t, scenario.TFOpts, "ssh_key_priv")
			sshUser := terraform.OutputRequired(t, scenario.TFOpts, "ssh_user")

			sshKeyPath := filepath.Join(scenario.ScenarioPath, "ssh_priv_cmd")
			err := os.WriteFile(sshKeyPath, []byte(sshKey), 0o600)
			require.NoError(t, err)
			// i dont know how to run interractive shell from tests as go tests are noniteractive
			// so it just prints the ssh command
			log.Info().Msg("-==SSH COMMAND HELPER==-")
			fmt.Printf("ssh -o IdentitiesOnly=yes -i %s %s@%s \n", sshKeyPath, sshUser, target)
		})
	}
}

func filter(t *testing.T, name string) {
	v, ok := os.LookupEnv("TF_TEST_STAGE")
	if ok {
		if v != name {
			t.Skip("Stage skipped due to TF_TEST_STAGE filter")
		}
	}
}

func utilFilter(name string) bool {
	v, ok := os.LookupEnv("TF_TEST_STAGE")
	if ok {
		if v == name {
			return true
		}
	}
	return false
}

func selectScenarios(scenarios []Scenario, filter string) ([]Scenario, error) {
	if filter == "" || filter == "all" {
		return scenarios, nil
	}

	var selected []Scenario
	for _, s := range scenarios {
		if s.Name == filter {
			selected = append(selected, s)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("TF_TEST_SCENARIO=%q matched no scenarios", filter)
	}
	return selected, nil
}

func validateStage(stage string) error {
	if stage == "" {
		return nil
	}
	switch stage {
	case "apply", "validate", "destroy", "build_scenario", "ssh":
		return nil
	default:
		return fmt.Errorf("unknown TF_TEST_STAGE=%q (want apply, validate, destroy, build_scenario, ssh)", stage)
	}
}

func sshRequested(stage, scenarioFilter, scenarioName string) bool {
	if stage != "ssh" {
		return false
	}
	if scenarioFilter == "" || scenarioFilter == "all" || scenarioFilter == scenarioName {
		return true
	}
	return false
}


func PathContainsIgnoredPath(path string, conf *Conf) bool {
	if conf == nil || len(conf.IgnoredPaths) == 0 {
		return false
	}

	cleanPath := filepath.Clean(path)
	sep := string(filepath.Separator)
	pathWithSeps := sep + cleanPath + sep

	for _, ignoredPath := range conf.IgnoredPaths {
		ignored := strings.TrimSpace(ignoredPath)
		if ignored == "" {
			continue
		}

		cleanIgnored := filepath.Clean(ignored)
		ignoredWithSeps := sep + cleanIgnored + sep
		if strings.Contains(pathWithSeps, ignoredWithSeps) {
			return true
		}
	}

	return false
}
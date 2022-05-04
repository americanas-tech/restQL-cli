package restql

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// Build generates a restQL binary using the given restQL version and the listed plugins.
func Build(pluginsInfo []string, restqlVersion string, restqlReplacement string, output string) error {
	absOutputFile, err := filepath.Abs(output)
	if err != nil {
		return err
	}

	plugins := make([]plugin, len(pluginsInfo))
	for i, pi := range pluginsInfo {
		plugins[i] = parsePluginInfo(pi)
	}

	tempDir, err := ioutil.TempDir("", "restql-compiling-*")
	if err != nil {
		return err
	}
	env := newEnvironment(tempDir, plugins, restqlVersion)
	if restqlReplacement != "" {
		env.UseRestqlReplacement(restqlReplacement)
	}

	err = env.Setup()
	if err != nil {
		return err
	}
	defer func() {
		cleanErr := env.Clean()
		if cleanErr != nil {
			logError("An error occurred when cleaning: %v", cleanErr)
		}
	}()

	err = runGoBuild(env, restqlVersion, absOutputFile)
	if err != nil {
		return err
	}

	return nil
}

func runGoBuild(env *environment, restqlVersion string, outputFile string) error {
	env.SetIfNotPresent("GOOS", "linux")
	env.SetIfNotPresent("CGO_ENABLED", 0)
	cmd := env.NewCommand("go", "build",
		"-o", outputFile,
		"-ldflags", fmt.Sprintf("-s -w -extldflags -static -X github.com/americanas-tech/restQL-golang/v4/cmd.build=%s", restqlVersion),
		"-tags", "netgo")

	err := env.RunCommand(cmd, ioutil.Discard)
	if err != nil {
		return err
	}

	return nil
}

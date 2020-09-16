package restql

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func Build(pluginsInfo []string, restqlVersion string, output string) error {
	absOutputFile, err := filepath.Abs(output)
	if err != nil {
		return err
	}

	plugins := make([]Plugin, len(pluginsInfo))
	for i, pi := range pluginsInfo {
		plugins[i] = ParsePluginInfo(pi)
	}

	tempDir, err := ioutil.TempDir("", "restql-compiling-*")
	if err != nil {
		return err
	}
	env := NewEnvironment(tempDir, plugins, restqlVersion)

	err = env.Setup()
	if err != nil {
		return err
	}
	defer func() {
		cleanErr := env.Clean()
		if cleanErr != nil {
			LogError("An error occurred when cleaning: %v", cleanErr)
		}
	}()

	err = runGoBuild(env, restqlVersion, absOutputFile)
	if err != nil {
		return err
	}

	return nil
}

func runGoBuild(env *Environment, restqlVersion string, outputFile string) error {
	env.SetIfNotPresent("GOOS", "linux")
	env.SetIfNotPresent("CGO_ENABLED", 0)
	cmd := env.NewCommand("go", "build",
		"-o", outputFile,
		"-ldflags", fmt.Sprintf("-s -w -extldflags -static -X github.com/b2wdigital/restQL-golang/v4/cmd.build=%s", restqlVersion),
		"-tags", "netgo")

	err := env.RunCommand(cmd, ioutil.Discard)
	if err != nil {
		return err
	}

	return nil
}

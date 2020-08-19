package restql

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
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

	err = runGoBuild(env, absOutputFile)
	if err != nil {
		return err
	}

	return nil
}

func runGoBuild(env *Environment, outputFile string) error {
	restqlVersion, err := getRestqlVersion(env)
	if err != nil {
		return err
	}

	env.SetIfNotPresent("GOOS", "linux")
	env.SetIfNotPresent("CGO_ENABLED", 0)
	cmd := env.NewCommand("go", "build",
		"-o", outputFile,
		"-ldflags", fmt.Sprintf("-s -w -extldflags -static -X main.build=%s", restqlVersion),
		"-tags", "netgo")

	err = env.RunCommand(cmd, ioutil.Discard)
	if err != nil {
		return err
	}

	return nil
}

func getRestqlVersion(env *Environment) (string, error) {
	var out bytes.Buffer
	cmd := env.NewCommand("go", "list", "-m", defaultRestqlModulePath)
	err := env.RunCommand(cmd, &out)
	if err != nil {
		return "", err
	}

	moduleNameAndVersion := strings.Split(out.String(), " ")
	if len(moduleNameAndVersion) < 2 {
		return "", errors.New("failed to fetch RestQL version from build environment")
	}

	return moduleNameAndVersion[1], nil
}

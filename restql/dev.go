package restql

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Run(restqlReplacement string, restqlVersion string, configLocation string, pluginLocation string, race bool) error {
	absPluginLocation, err := filepath.Abs(pluginLocation)
	if err != nil {
		return err
	}

	pluginDirective, err := getPlugin(absPluginLocation)
	if err != nil {
		return err
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	restqlEnvDir := filepath.Join(currentDir, "/.restql-env")

	env := NewEnvironment(restqlEnvDir, []Plugin{pluginDirective}, restqlVersion)
	if restqlReplacement != "" {
		env.UseRestqlReplacement(restqlReplacement)
	}

	if _, err := os.Stat(restqlEnvDir); os.IsNotExist(err) {
		err = env.Setup()
		if err != nil {
			return err
		}
	}

	if configLocation != "" {
		absConfigLocation, err := filepath.Abs(configLocation)
		if err != nil {
			return err
		}

		env.Set("RESTQL_CONFIG", absConfigLocation)
	}

	env.SetIfNotPresent("RESTQL_PORT", 9000)
	env.SetIfNotPresent("RESTQL_HEALTH_PORT", 9001)
	env.SetIfNotPresent("RESTQL_DEBUG_PORT", 9002)
	env.SetIfNotPresent("RESTQL_ENV", "development")

	cmd := env.NewCommand("go", "run", "main.go")
	if race {
		cmd.Args = append(cmd.Args, "-race")
	}

	err = env.RunCommand(cmd, os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func getPlugin(pluginLocation string) (Plugin, error) {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = pluginLocation
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return Plugin{}, fmt.Errorf("failed to execute command %v: %v: %s", cmd.Args, err, string(out))
	}
	currentPlugin := strings.TrimSpace(string(out))

	return Plugin{ModulePath: currentPlugin, Replace: pluginLocation}, nil
}
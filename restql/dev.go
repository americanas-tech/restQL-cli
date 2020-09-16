package restql

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run spin up a restQL instance using the given plugin.
//
// If a `pluginLocation` is not informed than the current directory is assumed.
// It inherit the environment variables and allow to set a custom restQL config and enable race detection.
// Also, it can use a different restQL source code with the `restqlReplacement`.
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

	env := newEnvironment(restqlEnvDir, []plugin{pluginDirective}, restqlVersion)
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

func getPlugin(pluginLocation string) (plugin, error) {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = pluginLocation
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return plugin{}, fmt.Errorf("failed to execute command %v: %v: %s", cmd.Args, err, string(out))
	}
	currentPlugin := strings.TrimSpace(string(out))

	return plugin{ModulePath: currentPlugin, Replace: pluginLocation}, nil
}

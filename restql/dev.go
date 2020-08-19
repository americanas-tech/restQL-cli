package restql

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Run(ctx context.Context, configLocation string, pluginLocation string, restqlVersion string) error {
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
	if _, err := os.Stat(restqlEnvDir); os.IsNotExist(err) {
		err = env.Setup(ctx)
		if err != nil {
			return err
		}
	}

	absConfigLocation, err := filepath.Abs(configLocation)
	if err != nil {
		return err
	}

	cmd := env.NewCommand("go", "run", "main.go")
	cmd.Env = setupRunningEnv(absConfigLocation)
	err = env.RunCommand(ctx, cmd, os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func setupRunningEnv(config string) []string {
	env := os.Environ()

	env = setIfNotPresent(env, "RESTQL_PORT", 9000)
	env = setIfNotPresent(env, "RESTQL_HEALTH_PORT", 9001)
	env = setIfNotPresent(env, "RESTQL_DEBUG_PORT", 9002)
	env = setIfNotPresent(env, "RESTQL_ENV", "development")

	if config != "" {
		for i, e := range env {
			if strings.HasPrefix("RESTQL_CONFIG=", e) {
				env[i] = fmt.Sprintf("RESTQL_CONFIG=%s", config)
			}
		}
		env = setIfNotPresent(env, "RESTQL_CONFIG", config)
	}

	return env
}

func setIfNotPresent(env []string, key string, defaultValue interface{}) []string {
	envVar := os.Getenv(key)
	if envVar == "" {
		return append(env, fmt.Sprintf("%s=%v", key, defaultValue))
	}
	return env
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
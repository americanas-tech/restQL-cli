package compilation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	restqlModulePath = "github.com/b2wdigital/restQL-golang"
)

func BuildRestQL(ctx context.Context, pluginsInfo []string, restqlVersion string, output string) error {
	absOutputFile, err := filepath.Abs(output)
	if err != nil {
		return err
	}

	plugins := make([]Plugin, len(pluginsInfo))
	for i, pi := range pluginsInfo {
		plugins[i] = ParsePluginInfo(pi)
	}

	env := NewEnvironment(plugins, restqlModulePath, restqlVersion)
	err = env.Setup(ctx)
	if err != nil {
		return err
	}
	defer func() {
		cleanErr := env.Clean()
		if cleanErr != nil {
			LogError("An error occurred when cleaning: %v", cleanErr)
		}
	}()

	err = runGoBuild(ctx, env, absOutputFile)
	if err != nil {
		return err
	}

	return nil
}

func runGoBuild(ctx context.Context, env *Environment, outputFile string) error {
	restqlVersion, err := getRestqlVersion(ctx, env)
	if err != nil {
		return err
	}

	cmd := env.NewCommand("go", "build",
		"-o", outputFile,
		"-ldflags", fmt.Sprintf("-s -w -extldflags -static -X main.build=%s", restqlVersion),
		"-tags", "netgo")
	cmd.Env = setupBuildEnv()

	err = env.RunCommand(ctx, cmd, ioutil.Discard)
	if err != nil {
		return err
	}

	return nil
}

func setupBuildEnv() []string {
	env := os.Environ()

	goos := os.Getenv("GOOS")
	if goos == "" {
		env = append(env, "GOOS=linux")
	}

	cgo := os.Getenv("CGO_ENABLED")
	if cgo == "" {
		env = append(env, "CGO_ENABLED=0")
	}

	return env
}

func getRestqlVersion(ctx context.Context, env *Environment) (string, error) {
	var out bytes.Buffer
	cmd := env.NewCommand("go", "list", "-m", restqlModulePath)
	err := env.RunCommand(ctx, cmd, &out)
	if err != nil {
		return "", err
	}

	moduleNameAndVersion := strings.Split(out.String(), " ")
	if len(moduleNameAndVersion) < 2 {
		return "", errors.New("failed to fetch RestQL version from build environment")
	}

	return moduleNameAndVersion[1], nil
}

var pluginInfoRegex = regexp.MustCompile("([^@=]+)@?([^=]*)=?(.*)")

type Plugin struct {
	ModulePath string
	Version    string
	Replace    string
}

func ParsePluginInfo(pluginInfo string) Plugin {
	if pluginInfo == "" {
		return Plugin{}
	}

	p := Plugin{}
	matches := pluginInfoRegex.FindAllStringSubmatch(pluginInfo, -1)
	if len(matches) < 1 {
		return Plugin{}
	}

	submatches := matches[0]
	if len(submatches) >= 2 {
		mn := submatches[1]
		p.ModulePath = mn
	}

	if len(submatches) >= 3 {
		v := submatches[2]
		p.Version = v
	}

	if len(submatches) >= 4 {
		r := submatches[3]
		p.Replace = r
	}

	return p
}



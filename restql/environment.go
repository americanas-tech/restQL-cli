package restql

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const defaultRestqlModulePath = "github.com/americanas-tech/restQL-golang"

const mainFileTemplate = `
package main

import (
	restqlcmd "{{ .RestqlModulePath }}/cmd"

	// add RestQL plugins here
	{{- range .Plugins}}
	_ "{{.}}"
	{{- end}}
)

func main() {
	restqlcmd.Start()
}
`

type environment struct {
	dir                 string
	vars                []string
	restqlModulePath    string
	restqlModuleVersion string
	restqlReplacement   string
	plugins             []plugin
}

func newEnvironment(dir string, plugins []plugin, restqlModuleVersion string) *environment {
	return &environment{
		dir:                 dir,
		vars:                os.Environ(),
		plugins:             plugins,
		restqlModulePath:    defaultRestqlModulePath,
		restqlModuleVersion: restqlModuleVersion,
	}
}

func (e *environment) Clean() error {
	return os.RemoveAll(e.dir)
}

func (e *environment) Set(key string, value interface{}) {
	prefix := fmt.Sprintf("%s=", key)
	newVar := fmt.Sprintf("%s=%v", key, value)

	for i, v := range e.vars {
		if strings.HasPrefix(v, prefix) {
			e.vars[i] = newVar
			return
		}
	}
	e.vars = append(e.vars, newVar)
}

func (e *environment) SetIfNotPresent(key string, value interface{}) {
	envVar := e.Get(key)
	if envVar == nil {
		e.vars = append(e.vars, fmt.Sprintf("%s=%v", key, value))
	}
}

func (e *environment) Get(key string) interface{} {
	prefix := fmt.Sprintf("%s=", key)
	for _, v := range e.vars {
		if strings.HasPrefix(v, prefix) {
			return v
		}
	}

	return nil
}

func (e *environment) GetAll() []string {
	return e.vars
}

func (e *environment) UseRestqlReplacement(path string) {
	e.restqlReplacement = path
}

func (e *environment) NewCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.dir
	cmd.Env = e.vars
	return cmd
}

func (e *environment) RunCommand(cmd *exec.Cmd, out io.Writer) error {
	logInfo("Executing command: %+v", cmd)

	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (e *environment) Setup() error {
	err := e.initializeDir()
	if err != nil {
		return err
	}

	err = e.setupMainFile()
	if err != nil {
		return err
	}

	err = e.setupGoMod()
	if err != nil {
		return err
	}

	err = e.setupDependenciesReplacements()
	if err != nil {
		return err
	}

	err = e.setupDependenciesVersions()
	if err != nil {
		return err
	}

	return nil
}

func (e *environment) initializeDir() error {
	if _, err := os.Stat(e.dir); os.IsNotExist(err) {
		return os.Mkdir(e.dir, 0700)
	}
	return nil
}

func (e *environment) setupMainFile() error {
	mainFileContent, err := parseMainFileTemplate(e)
	if err != nil {
		return err
	}

	mainFilePath := filepath.Join(e.dir, "main.go")
	logInfo("Writing main file to: %s", mainFilePath)
	err = ioutil.WriteFile(mainFilePath, mainFileContent, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (e *environment) setupGoMod() error {
	cmd := e.NewCommand("go", "mod", "init", "restql")
	err := e.RunCommand(cmd, ioutil.Discard)
	if err != nil {
		return err
	}

	return nil
}

func (e *environment) setupDependenciesReplacements() error {
	if e.restqlReplacement != "" {
		absReplacePath, err := filepath.Abs(e.restqlReplacement)
		if err != nil {
			return err
		}

		restqlMod, err := versionedModulePath(e.restqlModulePath, e.restqlModuleVersion)
		if err != nil {
			return err
		}

		logInfo("Replace dependency %s => %s", restqlMod, e.restqlReplacement)
		replaceArg := fmt.Sprintf("%s=%s", restqlMod, absReplacePath)

		cmd := e.NewCommand("go", "mod", "edit", "-replace", replaceArg)
		err = e.RunCommand(cmd, ioutil.Discard)
		if err != nil {
			return err
		}
	}

	for _, plugin := range e.plugins {
		if plugin.Replace == "" {
			continue
		}

		absReplacePath, err := filepath.Abs(plugin.Replace)
		if err != nil {
			return err
		}

		logInfo("Replace dependency %s => %s", plugin.ModulePath, plugin.Replace)
		replaceArg := fmt.Sprintf("%s=%s", plugin.ModulePath, absReplacePath)

		cmd := e.NewCommand("go", "mod", "edit", "-replace", replaceArg)
		err = e.RunCommand(cmd, ioutil.Discard)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *environment) setupDependenciesVersions() error {
	logInfo("Pinning versions")
	err := e.execGoGet(e.restqlModulePath, e.restqlModuleVersion)
	if err != nil {
		return err
	}

	for _, plugin := range e.plugins {
		err := e.execGoGet(plugin.ModulePath, plugin.Version)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *environment) execGoGet(modulePath, moduleVersion string) error {
	mod, err := versionedModulePath(modulePath, moduleVersion)
	if err != nil {
		return err
	}

	if moduleVersion != "" {
		mod += "@" + moduleVersion
	}
	cmd := e.NewCommand("go", "get", "-d", "-v", mod)
	return e.RunCommand(cmd, ioutil.Discard)
}

// versionedModulePath helps enforce Go Module's Semantic Import Versioning (SIV) by
// returning the form of modulePath with the major component of moduleVersion added,
// if > 1. For example, inputs of "foo" and "v1.0.0" will return "foo", but inputs
// of "foo" and "v2.0.0" will return "foo/v2", for use in Go imports and go commands.
// Inputs that conflict, like "foo/v2" and "v3.1.0" are an error. This function
// returns the input if the moduleVersion is not a valid semantic version string.
// If moduleVersion is empty string, the input modulePath is returned without error.
func versionedModulePath(modulePath, moduleVersion string) (string, error) {
	if moduleVersion == "" {
		return modulePath, nil
	}
	ver, err := semver.StrictNewVersion(strings.TrimPrefix(moduleVersion, "v"))
	if err != nil {
		// only return the error if we know they were trying to use a semantic version
		// (could have been a commit SHA or something)
		if strings.HasPrefix(moduleVersion, "v") {
			return "", fmt.Errorf("%s: %v", moduleVersion, err)
		}
		return modulePath, nil
	}
	major := ver.Major()

	// see if the module path has a major version at the end (SIV)
	matches := moduleVersionRegexp.FindStringSubmatch(modulePath)
	if len(matches) == 2 {
		modPathVer, err := strconv.Atoi(matches[1])
		if err != nil {
			return "", fmt.Errorf("this error should be impossible, but module path %s has bad version: %v", modulePath, err)
		}
		if modPathVer != int(major) {
			return "", fmt.Errorf("versioned module path (%s) and requested module major version (%d) diverge", modulePath, major)
		}
	} else if major > 1 {
		modulePath += fmt.Sprintf("/v%d", major)
	}

	return path.Clean(modulePath), nil
}

var moduleVersionRegexp = regexp.MustCompile(`.+/v(\d+)$`)

func parseMainFileTemplate(e *environment) ([]byte, error) {
	p := make([]string, len(e.plugins))
	for i, plugin := range e.plugins {
		p[i] = plugin.ModulePath
	}

	modPath, err := versionedModulePath(e.restqlModulePath, e.restqlModuleVersion)
	if err != nil {
		return nil, err
	}
	templateContext := mainFileTemplateContext{Plugins: p, RestqlModulePath: modPath}

	tpl, err := template.New("main").Parse(mainFileTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, templateContext)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type mainFileTemplateContext struct {
	RestqlModulePath string
	Plugins          []string
}

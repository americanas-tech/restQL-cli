package restql

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultRestqlModulePath = "github.com/b2wdigital/restQL-golang"

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

type Environment struct {
	dir                 string
	vars []string
	restqlModulePath    string
	restqlModuleVersion string
	plugins             []Plugin
}

func NewEnvironment(dir string, plugins []Plugin, restqlModuleVersion string) *Environment {
	return &Environment{
		dir: dir,
		vars: os.Environ(),
		plugins:             plugins,
		restqlModulePath:    defaultRestqlModulePath,
		restqlModuleVersion: restqlModuleVersion,
	}
}

func (e *Environment) Clean() error {
	return os.RemoveAll(e.dir)
}

func (e *Environment) Set(key string, value interface{}) {
	prefix := fmt.Sprintf("%s=", key)
	newVar := fmt.Sprintf("%s=%v", key, value)

	for i, v := range e.vars {
		if strings.HasPrefix(prefix, v) {
			e.vars[i] = newVar
			return
		}
	}
	e.vars = append(e.vars, newVar)
}

func (e *Environment) SetIfNotPresent(key string, value interface{}) {
	envVar := e.Get(key)
	if envVar == nil {
		e.vars = append(e.vars, fmt.Sprintf("%s=%v", key, value))
	}
}

func (e *Environment) Get(key string) interface{} {
	prefix := fmt.Sprintf("%s=", key)
	for _, v := range e.vars {
		if strings.HasPrefix(prefix, v) {
			return v
		}
	}

	return nil
}

func (e *Environment) GetAll() []string {
	return e.vars
}

func (e *Environment) NewCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.dir
	cmd.Env = e.vars
	return cmd
}

func (e *Environment) RunCommand(cmd *exec.Cmd, out io.Writer) error {
	LogInfo("Executing command: %+v", cmd)

	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (e *Environment) Setup() error {
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

func (e *Environment) initializeDir() error {
	if _, err := os.Stat(e.dir); os.IsNotExist(err) {
		return os.Mkdir(e.dir, 0700)
	}
	return nil
}

func (e *Environment) setupMainFile() error {
	mainFileContent, err := parseMainFileTemplate(e)
	if err != nil {
		return err
	}

	mainFilePath := filepath.Join(e.dir, "main.go")
	LogInfo("Writing main file to: %s", mainFilePath)
	err = ioutil.WriteFile(mainFilePath, mainFileContent, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (e *Environment) setupGoMod() error {
	cmd := e.NewCommand("go", "mod", "init", "restql")
	err := e.RunCommand(cmd, ioutil.Discard)
	if err != nil {
		return err
	}

	return nil
}

func (e *Environment) setupDependenciesReplacements() error {
	for _, plugin := range e.plugins {
		if plugin.Replace == "" {
			continue
		}

		absReplacePath, err := filepath.Abs(plugin.Replace)
		if err != nil {
			return err
		}

		LogInfo("Replace dependency %s => %s", plugin.ModulePath, plugin.Replace)
		replaceArg := fmt.Sprintf("%s=%s", plugin.ModulePath, absReplacePath)

		cmd := e.NewCommand("go", "mod", "edit", "-replace", replaceArg)
		err = e.RunCommand(cmd, ioutil.Discard)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Environment) setupDependenciesVersions() error {
	LogInfo("Pinning versions")
	err := e.execGoGet(e.restqlModulePath, e.restqlModuleVersion)
	if err != nil {
		return err
	}

	for _, plugin := range e.plugins {
		if plugin.Replace != "" {
			continue
		}

		err := e.execGoGet(plugin.ModulePath, plugin.Version)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Environment) execGoGet(modulePath, moduleVersion string) error {
	mod := modulePath
	if moduleVersion != "" {
		mod += "@" + moduleVersion
	}
	cmd := e.NewCommand("go", "get", "-d", "-v", mod)
	return e.RunCommand(cmd, ioutil.Discard)
}

func parseMainFileTemplate(e *Environment) ([]byte, error) {
	p := make([]string, len(e.plugins))
	for i, plugin := range e.plugins {
		p[i] = plugin.ModulePath
	}

	templateContext := mainFileTemplateContext{Plugins: p, RestqlModulePath: e.restqlModulePath}

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
	Plugins []string
}

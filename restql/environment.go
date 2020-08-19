package restql

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

func (e *Environment) RunCommand(ctx context.Context, cmd *exec.Cmd, out io.Writer) error {
	LogInfo("Executing command: %+v", cmd)

	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return err
	}

	cmdErrChan := make(chan error)
	go func() {
		cmdErrChan <- cmd.Wait()
	}()

	select {
	case cmdErr := <-cmdErrChan:
		return cmdErr
	case <-ctx.Done():
		select {
		case <-time.After(15 * time.Second):
			cmd.Process.Kill()
		case <-cmdErrChan:
		}
		return ctx.Err()
	}
}

func (e *Environment) Setup(ctx context.Context) error {
	err := e.initializeDir()
	if err != nil {
		return err
	}

	err = e.setupMainFile()
	if err != nil {
		return err
	}

	err = e.setupGoMod(ctx)
	if err != nil {
		return err
	}

	err = e.setupDependenciesReplacements(ctx)
	if err != nil {
		return err
	}

	err = e.setupDependenciesVersions(ctx)
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

func (e *Environment) setupGoMod(ctx context.Context) error {
	cmd := e.NewCommand("go", "mod", "init", "restql")
	err := e.RunCommand(ctx, cmd, ioutil.Discard)
	if err != nil {
		return err
	}

	return nil
}

func (e *Environment) setupDependenciesReplacements(ctx context.Context) error {
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
		err = e.RunCommand(ctx, cmd, ioutil.Discard)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Environment) setupDependenciesVersions(ctx context.Context) error {
	LogInfo("Pinning versions")
	err := e.execGoGet(ctx, e.restqlModulePath, e.restqlModuleVersion)
	if err != nil {
		return err
	}

	for _, plugin := range e.plugins {
		if plugin.Replace != "" {
			continue
		}

		err := e.execGoGet(ctx, plugin.ModulePath, plugin.Version)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Environment) execGoGet(ctx context.Context, modulePath, moduleVersion string) error {
	mod := modulePath
	if moduleVersion != "" {
		mod += "@" + moduleVersion
	}
	cmd := e.NewCommand("go", "get", "-d", "-v", mod)
	return e.RunCommand(ctx, cmd, ioutil.Discard)
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

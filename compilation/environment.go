package compilation

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
	"time"
)

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
	tempDir string
	restqlModulePath string
	restqlModuleVersion string
	plugins []Plugin
}

func NewEnvironment(plugins []Plugin, restqlModulePath string, restqlModuleVersion string) *Environment {
	return &Environment{
		plugins: plugins,
		restqlModulePath: restqlModulePath,
		restqlModuleVersion: restqlModuleVersion,
	}
}

func (e *Environment) Clean() error {
	return os.RemoveAll(e.tempDir)
}

func (e *Environment) Setup(ctx context.Context) error {
	tempDir, err := ioutil.TempDir("", "restql-compiling-*")
	if err != nil {
		return err
	}
	e.tempDir = tempDir

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

func (e *Environment) setupMainFile() error {
	mainFileContent, err := parseMainFileTemplate(e)
	if err != nil {
		return err
	}

	mainFilePath := filepath.Join(e.tempDir, "main.go")
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

func (e *Environment) NewCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.tempDir
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

package compilation

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const mainFileTemplate = `
package main

import (
	restqlcmd "github.com/b2wdigital/restQL-golang/cmd"

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
	plugins []Plugin
}

func NewEnvironment(plugins []Plugin) *Environment {
	return &Environment{plugins: plugins}
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
	cmd := e.newCommand("go", "mod", "init", "restql")
	err := e.runCommand(ctx, cmd, 1*time.Second)
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

		LogInfo("Replace dependency %s => %s", plugin.ModuleName, plugin.Replace)
		replaceArg := fmt.Sprintf("%s=%s", plugin.ModuleName, plugin.Replace)

		cmd := e.newCommand("go", "mod", "edit", "-replace", replaceArg)
		err := e.runCommand(ctx, cmd, 1*time.Second)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Environment) newCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Dir = e.tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (e *Environment) runCommand(ctx context.Context, cmd *exec.Cmd, timeout time.Duration) error {
	LogInfo("Executing command (timeout=%s): %+v", timeout, cmd)

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

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
		p[i] = plugin.ModuleName
	}

	templateContext := mainFileTemplateContext{Plugins: p}

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
	Plugins []string
}

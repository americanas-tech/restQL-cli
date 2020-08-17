package compilation

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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

func (e *Environment) Setup() error {
	tempDir, err := ioutil.TempDir("", "restql-compiling-*")
	if err != nil {
		return err
	}
	e.tempDir = tempDir

	err = e.setupMainFile()
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
	log.Printf("[INFO] Writing main file to: %s\n", mainFilePath)
	err = ioutil.WriteFile(mainFilePath, mainFileContent, 0644)
	if err != nil {
		return err
	}
	return nil
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

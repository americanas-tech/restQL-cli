package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	app := NewApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("[ERROR] failed to initialize RestQL CLI : %v", err)
		os.Exit(1)
	}
}

func NewApp() *cli.App {
	return &cli.App{
		Name: "restql",
		Usage: "Builds custom binaries for RestQL with the given plugins",
		Action: func(ctx *cli.Context) error {
			fmt.Println("Hello World")
			return nil
		},
	}
}

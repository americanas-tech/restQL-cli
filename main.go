package main

import (
	"fmt"
	"github.com/b2wdigital/restQL-golang-cli/compilation"
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
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name: "with",
				Aliases: []string{"w"},
				Required: true,
				Usage: "Specify the Go Module name of the plugin, can optionally set the version and a replace path: github.com/user/plugin[@version][=../replace/path]",
			},
			&cli.StringFlag{
				Name: "output",
				Aliases: []string{"o"},
				Value: "./",
				Usage: "Set the location where the final binary will be placed",
			},
		},
		Action: func(ctx *cli.Context) error {
			withPlugins := ctx.StringSlice("with")
			output := ctx.String("output")

			restqlVersion := ctx.Args().Get(0)

			return compilation.BuildRestQL(ctx.Context, withPlugins, restqlVersion, output)
		},
	}
}

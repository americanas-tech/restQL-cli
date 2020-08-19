package main

import (
	"fmt"
	"github.com/b2wdigital/restQL-golang-cli/restql"
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
		Usage: "Manage the development and building of plugins within RestQL",
		Commands: []*cli.Command{
			{
				Name: "build",
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

					return restql.Build(ctx.Context, withPlugins, restqlVersion, output)
				},
			},
			{
				Name: "run",
				Usage: "Run RestQL with the plugin at working directory",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "config",
						Aliases: []string{"c"},
						Value: "./restql.yml",
						Usage: "Set the location where the YAML configuration file is placed",
					},
					&cli.StringFlag{
						Name: "plugin",
						Aliases: []string{"p"},
						Value: "./",
						Usage: "Set the location of the plugin in development",
					},
					&cli.BoolFlag{
						Name: "race",
						Value: false,
						Usage: "Enable Go race detection",
					},
				},
				Action: func(ctx *cli.Context) error {
					config := ctx.String("config")
					pluginLocation := ctx.String("plugin")
					race := ctx.Bool("race")
					restqlVersion := ctx.Args().Get(0)

					return restql.Run(ctx.Context, restqlVersion, config, pluginLocation, race)
				},
			},
		},

	}
}

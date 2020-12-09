# RestQL CLI

This command line tool makes it easy to develop plugins and build custom [RestQL](https://github.com/b2wdigital/restQL-golang) binaries with plugins linked.

> Note: this tool only works with the Golang version of RestQL.

## Requirements 
[Go installed](https://golang.org/doc/install)

## Installation
Provided you have Go correctly installed with the `$GOPATH` and `$GOBIN` environment variables set, run:
```shell script
$ go get -u github.com/b2wdigital/restQL-cli
```

This will install or update the package, and the `restQL-cli` command will be installed in your `$GOBIN` directory.

## Usage
This tool supports two commands, `build` and `run`. Both depend on Go Modules, so any plugin must by a Go Module compatible package.

As usual in the Go platform, this tool passes-through all environment variables set by the user. It only provides defaults for some of them and any exception will be specified ahead. 

### Running

If you are in the plugin's directory you can start a RestQL instance with your plugin using the following command:
```shell script
$ restQL-cli run --config=/path/to/file
```
The `--config` flag indicates where the [RestQL YAML configuration file](https://golang.org/doc/articles/race_detector.html) to be used is placed. It takes precedence over the `RESTQL_CONFIG` environment variable.

Alternatively, you can run this command from anywhere and point to the plugin with the `--plugin` flag.

You can also replace the restQL source code used to spin up the instance using the `--restql-replacement` flag.

In the first time it is run it will create a hidden directory `.restql-env` which will have set up for executing RestQL together with your plugin.

If you make any changes to your plugin, just restart the command, it will pick-up the current version and avoid rebuilding the environment folder. 

This tool also provides the ability to enable the Go race detector during developing, you can enable it using the `--race` flag.

### Building

When building a custom binary you can specify as many plugins as you wish using their module name, same as you would use for when running `go get`, for example:
```shell script
$ restQL-cli build --with github.com/user/plugin-a --output ./custom-restQL
```

This will create a binary in the specified `output` path.

The `--with` flag accepts a string with the format `github.com/user/plugin-a[@version][=path/to/replacement]`.

You can also replace the restQL source code to be used with the `--restql-replacement` flag.

## License

The [MIT license](https://mit-license.org/). See the LICENSE file.


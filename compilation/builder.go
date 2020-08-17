package compilation

import (
	"context"
	"fmt"
	"regexp"
)

func BuildRestQL(ctx context.Context, pluginsInfo []string, output string) error {
	plugins := make([]Plugin, len(pluginsInfo))
	for i, pi := range pluginsInfo {
		plugins[i] = ParsePluginInfo(pi)
	}

	env := NewEnvironment(plugins)
	err := env.Setup(ctx)
	if err != nil {
		return err
	}
	//defer env.Clean()

	fmt.Printf("Build restql with: %v, to output: %s\n", pluginsInfo, output)
	return nil
}

var pluginInfoRegex = regexp.MustCompile("([^@=]+)@?([^=]*)=?(.*)")

type Plugin struct {
	ModuleName string
	Version string
	Replace string
}

func ParsePluginInfo(pluginInfo string) Plugin {
	if pluginInfo == "" {
		return Plugin{}
	}

	p := Plugin{}
	matches := pluginInfoRegex.FindAllStringSubmatch(pluginInfo, -1)
	if len(matches) < 1 {
		return Plugin{}
	}

	submatches := matches[0]
	if len(submatches) >= 2 {
		mn := submatches[1]
		p.ModuleName = mn
	}

	if len(submatches) >= 3 {
		v := submatches[2]
		p.Version = v
	}

	if len(submatches) >= 4 {
		r := submatches[3]
		p.Replace = r
	}

	return p
}



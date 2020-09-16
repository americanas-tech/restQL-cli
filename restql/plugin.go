package restql

import "regexp"

var pluginInfoRegex = regexp.MustCompile("([^@=]+)@?([^=]*)=?(.*)")

type plugin struct {
	ModulePath string
	Version    string
	Replace    string
}

func parsePluginInfo(pluginInfo string) plugin {
	if pluginInfo == "" {
		return plugin{}
	}

	p := plugin{}
	matches := pluginInfoRegex.FindAllStringSubmatch(pluginInfo, -1)
	if len(matches) < 1 {
		return plugin{}
	}

	submatches := matches[0]
	if len(submatches) >= 2 {
		mn := submatches[1]
		p.ModulePath = mn
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

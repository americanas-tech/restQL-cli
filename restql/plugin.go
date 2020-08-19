package restql

import "regexp"

var pluginInfoRegex = regexp.MustCompile("([^@=]+)@?([^=]*)=?(.*)")

type Plugin struct {
	ModulePath string
	Version    string
	Replace    string
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

package restql

import (
	"github.com/b2wdigital/restQL-golang-cli/test"
	"testing"
)

func TestParsePluginInfo(t *testing.T) {
	tests := []struct{
		name     string
		input    string
		expected Plugin
	}{
		{
			"when given an empty string, return an empty plugin",
			"",
			Plugin{},
		},
		{
			"when given an plugin info with only the module name return an plugin with it",
			"github.com/user/plugin",
			Plugin{
				ModulePath: "github.com/user/plugin",
			},
		},
		{
			"when given an plugin info with the module name and version return an plugin with they",
			"github.com/user/plugin@1.9.0",
			Plugin{
				ModulePath: "github.com/user/plugin",
				Version:    "1.9.0",
			},
		},
		{
			"when given an plugin info with the module name and replace path return an plugin with they",
			"github.com/user/plugin=../replace/path",
			Plugin{
				ModulePath: "github.com/user/plugin",
				Replace:    "../replace/path",
			},
		},
		{
			"when given an plugin info with the module name, version and replace path return an plugin with they",
			"github.com/user/plugin@1.9.0=../replace/path",
			Plugin{
				ModulePath: "github.com/user/plugin",
				Version:    "1.9.0",
				Replace:    "../replace/path",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePluginInfo(tt.input)
			test.Equal(t, got, tt.expected)
		})
	}
}

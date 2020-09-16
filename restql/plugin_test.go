package restql

import (
	"reflect"
	"testing"
)

func TestParsePluginInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected plugin
	}{
		{
			"when given an empty string, return an empty plugin",
			"",
			plugin{},
		},
		{
			"when given an plugin info with only the module name return an plugin with it",
			"github.com/user/plugin",
			plugin{
				ModulePath: "github.com/user/plugin",
			},
		},
		{
			"when given an plugin info with the module name and version return an plugin with they",
			"github.com/user/plugin@1.9.0",
			plugin{
				ModulePath: "github.com/user/plugin",
				Version:    "1.9.0",
			},
		},
		{
			"when given an plugin info with the module name and replace path return an plugin with they",
			"github.com/user/plugin=../replace/path",
			plugin{
				ModulePath: "github.com/user/plugin",
				Replace:    "../replace/path",
			},
		},
		{
			"when given an plugin info with the module name, version and replace path return an plugin with they",
			"github.com/user/plugin@1.9.0=../replace/path",
			plugin{
				ModulePath: "github.com/user/plugin",
				Version:    "1.9.0",
				Replace:    "../replace/path",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePluginInfo(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("got = %+#v, want = %+#v", got, tt.expected)
			}
		})
	}
}

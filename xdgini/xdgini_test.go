package xdgini_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qnighy/wsl-open-proxy/xdgini"
)

func TestParseStringifyRoundtrip(t *testing.T) {
	testcases := []struct {
		name  string
		input string
	}{
		{
			name:  "empty",
			input: "",
		},
		{
			name:  "simple",
			input: "[Foo]\nKey1=Value1\nKey2=Value2\n[Bar]\nKey3=Value3\n",
		},
		{
			name:  "without last newline",
			input: "[Foo]\nKey1=Value1\nKey2=Value2\n[Bar]\nKey3=Value3",
		},
		{
			name:  "with comments and empty lines",
			input: "\n# Comment 1\n\n[Foo]\n\n# Comment 2\n\nKey1=Value1\n\n# Comment 3\n\nKey2=Value2\n\n# Comment 4\n\n[Bar]\n\n# Comment 5\n\nKey3=Value3\n\n# Comment 6\n\n",
		},
		{
			name:  "with dummy groups",
			input: "Key1=Value1\n",
		},
		{
			name:  "with broken group name",
			input: "[Foo\nKey1=Value1\n",
		},
		{
			name:  "with broken key-value pair",
			input: "[Foo]\nBar\n",
		},
		{
			name:  "with trailing spaces",
			input: "[Foo] \n Key1 = Value1 \n \n",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			config := xdgini.ParseConfig(tc.input)
			output := config.String()
			if diff := cmp.Diff(tc.input, output); diff != "" {
				t.Errorf("ParseConfig(String()) mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

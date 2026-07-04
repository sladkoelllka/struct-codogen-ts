package parser

import (
	"path/filepath"
	"testing"
)

func TestExtractEntityFromDirKeepsNestedHandlerPath(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{
			name: "flat dto",
			dir:  filepath.Join("internal", "delivery", "handler", "report", "dto"),
			want: "report",
		},
		{
			name: "nested dto",
			dir:  filepath.Join("internal", "delivery", "handler", "report", "whitebox", "dto"),
			want: "report/whitebox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractEntityFromDir(tt.dir); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

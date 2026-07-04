package permissions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePath(t *testing.T) {
	dir := t.TempDir()

	content := `package permission

import "gitlab.legion.devel/legion/helpdesk-server/pkg/authz"

const (
	UsersRead   authz.Permission = "users.read"
	UsersManage authz.Permission = "users.manage"
	Ignored     string           = "ignored"
)
`

	err := os.WriteFile(filepath.Join(dir, "permission.go"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	permissions, err := ParsePath(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []Permission{
		{Name: "UsersRead", Value: "users.read"},
		{Name: "UsersManage", Value: "users.manage"},
	}

	if len(permissions) != len(expected) {
		t.Fatalf("expected %d permissions, got %d", len(expected), len(permissions))
	}

	for i, permission := range expected {
		if permissions[i] != permission {
			t.Fatalf("expected permission %v at index %d, got %v", permission, i, permissions[i])
		}
	}
}

func TestGenerate(t *testing.T) {
	output := Generate("Permission", []Permission{
		{Name: "UsersRead", Value: "users.read"},
		{Name: "UsersManage", Value: "users.manage"},
	})

	expectedParts := []string{
		`export const Permission = {`,
		`UsersRead: "users.read",`,
		`UsersManage: "users.manage",`,
		`} as const;`,
		`export type Permission = typeof Permission[keyof typeof Permission];`,
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("expected output to contain %q, got:\n%s", part, output)
		}
	}
}

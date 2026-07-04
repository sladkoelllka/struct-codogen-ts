package errorcodes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePath(t *testing.T) {
	dir := t.TempDir()

	content := `package apperrs

import "github.com/sladkoelllka/apihandler/errs"

const (
	CodeUserNotFound errs.ErrorCode = "USER_NOT_FOUND"
	CodeEmailTaken   = "EMAIL_TAKEN"
	CodeFileTooLarge errs.ErrorCode = "FILE_TOO_LARGE"
)

var (
	UserNotFound = errs.Type{Code: CodeUserNotFound, HTTPStatus: 404}
	EmailTaken   = errs.Type{Code: CodeEmailTaken, HTTPStatus: 409}
	FileTooLarge = errs.Type{Code: CodeFileTooLarge, HTTPStatus: 413}
)
`

	err := os.WriteFile(filepath.Join(dir, "errors.go"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	codes, err := ParsePath(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(codes) != 3 {
		t.Fatalf("expected 3 codes, got %d", len(codes))
	}

	expected := []Code{
		{Name: "UserNotFound", Value: "USER_NOT_FOUND"},
		{Name: "EmailTaken", Value: "EMAIL_TAKEN"},
		{Name: "FileTooLarge", Value: "FILE_TOO_LARGE"},
	}

	for i, code := range expected {
		if codes[i] != code {
			t.Fatalf("expected code %v at index %d, got %v", code, i, codes[i])
		}
	}
}

func TestGenerate(t *testing.T) {
	output := Generate([]Code{
		{Name: "UserNotFound", Value: "USER_NOT_FOUND"},
		{Name: "EmailTaken", Value: "EMAIL_TAKEN"},
		{Name: "FileTooLarge", Value: "FILE_TOO_LARGE"},
	})

	expectedParts := []string{
		`export const ErrorCode = {`,
		`ErrBadRequest: "BAD_REQUEST",`,
		`ErrUnauthorized: "UNAUTHORIZED",`,
		`ErrForbidden: "FORBIDDEN",`,
		`ErrNotFound: "NOT_FOUND",`,
		`ErrConflict: "CONFLICT",`,
		`ErrTooManyRequests: "TOO_MANY_REQUESTS",`,
		`ErrInternal: "INTERNAL_SERVER_ERROR",`,
		`UserNotFound: "USER_NOT_FOUND",`,
		`EmailTaken: "EMAIL_TAKEN",`,
		`FileTooLarge: "FILE_TOO_LARGE",`,
		`} as const;`,
		`export type ErrorCode = typeof ErrorCode[keyof typeof ErrorCode];`,
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("expected output to contain %q, got:\n%s", part, output)
		}
	}
}

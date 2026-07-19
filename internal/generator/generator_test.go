package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sladkoelllka/struct-codogen-ts/internal/parser"
)

func TestGenerateEmbeddedStruct(t *testing.T) {
	tmp, err := os.CreateTemp("", "embedded_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString(`package dto

type payload struct {
	FirstName  string ` + "`json:\"first_name\"`" + `
	LastName   string ` + "`json:\"last_name\"`" + `
}

type RequestUser struct {
	ID int ` + "`json:\"id\"`" + `
	payload
}
`)
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	file, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	g := NewGenerator(GeneratorConfig{ExportType: "interface"})
	out := g.generateFileContent(file)

	if !strings.Contains(out, "export interface RequestUser extends payload") {
		t.Fatalf("expected RequestUser to extend payload, got:\n%s", out)
	}

	if !strings.Contains(out, "id: number;") {
		t.Fatalf("expected id field in RequestUser, got:\n%s", out)
	}

	if !strings.Contains(out, "export interface payload") {
		t.Fatalf("expected payload interface generated, got:\n%s", out)
	}
}

func TestGenerateAliasType(t *testing.T) {
	tmp, err := os.CreateTemp("", "alias_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString(`package dto

type LoginResponse struct {
	Token string ` + "`json:\"token\"`" + `
}

type CreateAdminResponse = LoginResponse
`)
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	file, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	g := NewGenerator(GeneratorConfig{ExportType: "interface"})
	out := g.generateFileContent(file)

	if !strings.Contains(out, "export type CreateAdminResponse = LoginResponse;") {
		t.Fatalf("expected alias type generated, got:\n%s", out)
	}
}

func TestGenerateJSONOmitEmptyField(t *testing.T) {
	tmp, err := os.CreateTemp("", "omitempty_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString(`package dto

type Response struct {
	Required string ` + "`json:\"required\"`" + `
	Optional int64  ` + "`json:\"optional,omitempty\"`" + `
}
`)
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	file, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	g := NewGenerator(GeneratorConfig{ExportType: "interface"})
	out := g.generateFileContent(file)

	if !strings.Contains(out, "required: string;") {
		t.Fatalf("expected required field to stay required, got:\n%s", out)
	}
	if !strings.Contains(out, "optional?: number;") {
		t.Fatalf("expected omitempty field to be optional, got:\n%s", out)
	}
}

func TestGenerateMapTypesAsRecord(t *testing.T) {
	tmp, err := os.CreateTemp("", "maps_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString(`package dto

type Child struct {
	Name string ` + "`json:\"name\"`" + `
}

type Response struct {
	Meta     map[string]int64 ` + "`json:\"meta\"`" + `
	Children map[string][]*Child ` + "`json:\"children\"`" + `
	Nested   map[string]map[string]string ` + "`json:\"nested\"`" + `
}
`)
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	file, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	g := NewGenerator(GeneratorConfig{ExportType: "interface"})
	out := g.generateFileContent(file)

	expected := []string{
		`meta: Record<string, number>;`,
		`children: Record<string, Child[]>;`,
		`nested: Record<string, Record<string, string>>;`,
	}

	for _, snippet := range expected {
		if !strings.Contains(out, snippet) {
			t.Fatalf("expected output to contain %q, got:\n%s", snippet, out)
		}
	}
}

func TestGenerateNamedStringTypes(t *testing.T) {
	tmp, err := os.CreateTemp("", "named_string_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString(`package dto

type FindingsFilter struct {
	HostIP    *string               ` + "`form:\"host_ip\"`" + `
	OrderBy   *FindingFilterOrderBy ` + "`form:\"order_by\"`" + `
	SortOrder *SortOrder           ` + "`form:\"sort_order\"`" + `
}

type FindingFilterOrderBy string

const (
	FindingFilterOrderByHostIP  FindingFilterOrderBy = "ip"
	FindingFilterOrderByHostMac FindingFilterOrderBy = "mac"
)

type SortOrder string
`)
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	file, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	g := NewGenerator(GeneratorConfig{ExportType: "interface"})
	out := g.generateFileContent(file)

	expected := []string{
		`export interface FindingsFilter {`,
		`host_ip?: string;`,
		`order_by?: FindingFilterOrderBy;`,
		`sort_order?: SortOrder;`,
		`export type FindingFilterOrderBy = "ip" | "mac";`,
		`export type SortOrder = string;`,
	}

	for _, snippet := range expected {
		if !strings.Contains(out, snippet) {
			t.Fatalf("expected output to contain %q, got:\n%s", snippet, out)
		}
	}
}

func TestGeneratePerFileAliasImportsSelectorType(t *testing.T) {
	g := NewGenerator(GeneratorConfig{ExportType: "interface"})

	files := []*parser.File{
		{
			Entity:   "admin",
			Type:     "response",
			Generate: true,
			Imports: map[string]string{
				"dto": "gitlab.legion.devel/legion/helpdesk-server/internal/delivery/handler/auth/dto",
			},
			Structs: []parser.Struct{
				{Name: "CreateAdminResponse", Package: "dto", AliasType: "dto.LoginResponse"},
			},
		},
		{
			Entity:   "auth",
			Type:     "login",
			Generate: false,
			Structs: []parser.Struct{
				{
					Name:    "LoginResponse",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "Token", Type: "string", JSONTag: "token"},
					},
				},
			},
		},
	}

	outputs := g.GeneratePerFile(files, "/tmp/generated")

	var responseContent string
	for _, output := range outputs {
		if output.Path == filepath.Join("/tmp/generated", "admin", "response.ts") {
			responseContent = output.Content
			break
		}
	}

	if !strings.Contains(responseContent, "import type { LoginResponse } from '../auth/login';") {
		t.Fatalf("expected selector alias import, got:\n%s", responseContent)
	}

	if !strings.Contains(responseContent, "export type CreateAdminResponse = LoginResponse;") {
		t.Fatalf("expected selector alias export, got:\n%s", responseContent)
	}
}

func TestGeneratePerFileMapValueImportsSelectorType(t *testing.T) {
	g := NewGenerator(GeneratorConfig{ExportType: "interface"})

	files := []*parser.File{
		{
			Entity:   "admin",
			Type:     "response",
			Generate: true,
			Imports: map[string]string{
				"dto": "gitlab.legion.devel/legion/helpdesk-server/internal/delivery/handler/auth/dto",
			},
			Structs: []parser.Struct{
				{
					Name:    "AdminResponse",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "Sessions", Type: "map[string]dto.LoginResponse", JSONTag: "sessions"},
					},
				},
			},
		},
		{
			Entity:   "auth",
			Type:     "login",
			Generate: false,
			Structs: []parser.Struct{
				{
					Name:    "LoginResponse",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "Token", Type: "string", JSONTag: "token"},
					},
				},
			},
		},
	}

	outputs := g.GeneratePerFile(files, "/tmp/generated")

	var responseContent string
	for _, output := range outputs {
		if output.Path == filepath.Join("/tmp/generated", "admin", "response.ts") {
			responseContent = output.Content
			break
		}
	}

	if !strings.Contains(responseContent, "import type { LoginResponse } from '../auth/login';") {
		t.Fatalf("expected selector map value import, got:\n%s", responseContent)
	}
	if !strings.Contains(responseContent, "sessions: Record<string, LoginResponse>;") {
		t.Fatalf("expected selector map value to be generated as Record, got:\n%s", responseContent)
	}
	if strings.Contains(responseContent, "dto.LoginResponse") {
		t.Fatalf("unexpected Go selector in generated TypeScript, got:\n%s", responseContent)
	}
}

func TestGeneratePerFileAliasImportsReadmodelType(t *testing.T) {
	g := NewGenerator(GeneratorConfig{ExportType: "interface"})

	files := []*parser.File{
		{
			Entity:   "user",
			Type:     "response",
			Generate: true,
			Imports: map[string]string{
				"readmodel": "gitlab.legion.devel/legion/helpdesk-server/internal/readmodel",
			},
			Structs: []parser.Struct{
				{Name: "UserResponse", Package: "dto", AliasType: "readmodel.User"},
			},
		},
		{
			Entity:   "readmodel",
			Type:     "user",
			Generate: true,
			Structs: []parser.Struct{
				{
					Name:    "User",
					Package: "readmodel",
					Fields: []parser.Field{
						{Name: "ID", Type: "int64", JSONTag: "id"},
						{Name: "DisplayName", Type: "string", JSONTag: "display_name"},
					},
				},
			},
		},
	}

	outputs := g.GeneratePerFile(files, "/tmp/generated")

	contents := make(map[string]string, len(outputs))
	for _, output := range outputs {
		contents[output.Path] = output.Content
	}

	responsePath := filepath.Join("/tmp/generated", "user", "response.ts")
	readmodelPath := filepath.Join("/tmp/generated", "readmodel", "user.ts")

	if !strings.Contains(contents[responsePath], "import type { User } from '../readmodel/user';") {
		t.Fatalf("expected readmodel import, got:\n%s", contents[responsePath])
	}
	if !strings.Contains(contents[responsePath], "export type UserResponse = User;") {
		t.Fatalf("expected alias to imported readmodel type, got:\n%s", contents[responsePath])
	}
	if !strings.Contains(contents[readmodelPath], "display_name: string;") {
		t.Fatalf("expected readmodel JSON-tagged field, got:\n%s", contents[readmodelPath])
	}
}

func TestGeneratePerFileImportsEmbeddedSelectorType(t *testing.T) {
	g := NewGenerator(GeneratorConfig{ExportType: "interface"})

	files := []*parser.File{
		{
			Entity:   "pentest",
			Type:     "summary",
			Generate: true,
			Imports: map[string]string{
				"dto": "gitlab.legion.devel/legion/helpdesk-server/internal/delivery/handler/report/dto",
			},
			Structs: []parser.Struct{
				{
					Name:    "PentestSummary",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "dto.ReportSummary", Type: "dto.ReportSummary", Embedded: true},
						{Name: "Findings", Type: "int64", JSONTag: "findings"},
						{Name: "Severity", Type: "ReportSeveritySummary", JSONTag: "severity"},
					},
				},
				{
					Name:    "ReportSeveritySummary",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "Critical", Type: "int64", JSONTag: "critical"},
					},
				},
			},
		},
		{
			Entity:   "report",
			Type:     "summary",
			Generate: false,
			Structs: []parser.Struct{
				{
					Name:    "ReportSummary",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "ID", Type: "int64", JSONTag: "id"},
					},
				},
			},
		},
	}

	outputs := g.GeneratePerFile(files, "/tmp/generated")

	var summaryContent string
	for _, output := range outputs {
		if output.Path == filepath.Join("/tmp/generated", "pentest", "summary.ts") {
			summaryContent = output.Content
			break
		}
	}

	if !strings.Contains(summaryContent, "import type { ReportSummary } from '../report/summary';") {
		t.Fatalf("expected embedded selector import, got:\n%s", summaryContent)
	}
	if !strings.Contains(summaryContent, "export interface PentestSummary extends ReportSummary {") {
		t.Fatalf("expected embedded selector to extend imported TS type, got:\n%s", summaryContent)
	}
	if strings.Contains(summaryContent, "dto.ReportSummary") {
		t.Fatalf("unexpected Go selector in generated TypeScript, got:\n%s", summaryContent)
	}
}

func TestGeneratePerFileAliasesSameNameSelectorType(t *testing.T) {
	g := NewGenerator(GeneratorConfig{ExportType: "interface"})

	files := []*parser.File{
		{
			Entity:   "provider",
			Type:     "response",
			Generate: true,
			Imports: map[string]string{
				"dto": "gitlab.legion.devel/legion/helpdesk-server/internal/delivery/handler/user/dto",
			},
			Structs: []parser.Struct{
				{
					Name:    "ProviderRoleMappingResponse",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "ID", Type: "int64", JSONTag: "id"},
						{Name: "Role", Type: "RoleRef", JSONTag: "role"},
					},
				},
				{Name: "RoleRef", Package: "dto", AliasType: "dto.RoleRef"},
			},
		},
		{
			Entity:   "user",
			Type:     "user",
			Generate: false,
			Structs: []parser.Struct{
				{
					Name:    "RoleRef",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "ID", Type: "int64", JSONTag: "id"},
						{Name: "Name", Type: "string", JSONTag: "name"},
					},
				},
			},
		},
	}

	outputs := g.GeneratePerFile(files, "/tmp/generated")

	var responseContent string
	for _, output := range outputs {
		if output.Path == filepath.Join("/tmp/generated", "provider", "response.ts") {
			responseContent = output.Content
			break
		}
	}

	if !strings.Contains(responseContent, "import type { RoleRef as DtoRoleRef } from '../user/user';") {
		t.Fatalf("expected same-name selector alias import, got:\n%s", responseContent)
	}

	if !strings.Contains(responseContent, "role: RoleRef;") {
		t.Fatalf("expected field to use local RoleRef alias, got:\n%s", responseContent)
	}

	if !strings.Contains(responseContent, "export type RoleRef = DtoRoleRef;") {
		t.Fatalf("expected non-self-referential alias export, got:\n%s", responseContent)
	}
}

func TestGeneratePerFileAliasesSameNameReadmodelType(t *testing.T) {
	g := NewGenerator(GeneratorConfig{ExportType: "interface"})

	files := []*parser.File{
		{
			Entity:   "ticket",
			Type:     "request",
			Generate: true,
			Imports: map[string]string{
				"readmodel": "gitlab.legion.devel/legion/helpdesk-server/internal/readmodel",
			},
			Structs: []parser.Struct{
				{Name: "Filter", Package: "dto", AliasType: "readmodel.Filter"},
			},
		},
		{
			Entity:   "readmodel",
			Type:     "filter",
			Generate: true,
			Structs: []parser.Struct{
				{
					Name:    "Filter",
					Package: "readmodel",
					Fields: []parser.Field{
						{Name: "Status", Type: "string", JSONTag: "status"},
					},
				},
			},
		},
	}

	outputs := g.GeneratePerFile(files, "/tmp/generated")

	var requestContent string
	for _, output := range outputs {
		if output.Path == filepath.Join("/tmp/generated", "ticket", "request.ts") {
			requestContent = output.Content
			break
		}
	}

	if !strings.Contains(requestContent, "import type { Filter as ReadmodelFilter } from '../readmodel/filter';") {
		t.Fatalf("expected same-name readmodel alias import, got:\n%s", requestContent)
	}
	if !strings.Contains(requestContent, "export type Filter = ReadmodelFilter;") {
		t.Fatalf("expected alias to renamed readmodel type, got:\n%s", requestContent)
	}
	if strings.Contains(requestContent, "export type Filter = Filter;") {
		t.Fatalf("unexpected self-referential alias, got:\n%s", requestContent)
	}
}

func TestGeneratePerFileKeepsNestedEntityOutputPath(t *testing.T) {
	g := NewGenerator(GeneratorConfig{ExportType: "interface"})

	files := []*parser.File{
		{
			Entity:   "report",
			Type:     "request",
			Generate: true,
			Structs: []parser.Struct{
				{
					Name:    "Report",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "Bang", Type: "string", JSONTag: "bang"},
					},
				},
			},
		},
		{
			Entity:   "report/whitebox",
			Type:     "request",
			Generate: true,
			Structs: []parser.Struct{
				{
					Name:    "Request",
					Package: "dto",
					Fields: []parser.Field{
						{Name: "Name", Type: "string", JSONTag: "name"},
					},
				},
			},
		},
	}

	outputs := g.GeneratePerFile(files, "/tmp/generated")

	contents := make(map[string]string, len(outputs))
	for _, output := range outputs {
		contents[output.Path] = output.Content
	}

	reportPath := filepath.Join("/tmp/generated", "report", "request.ts")
	whiteboxPath := filepath.Join("/tmp/generated", "report", "whitebox", "request.ts")

	if !strings.Contains(contents[reportPath], "export interface Report") {
		t.Fatalf("expected report request output at %s, got:\n%s", reportPath, contents[reportPath])
	}
	if !strings.Contains(contents[whiteboxPath], "export interface Request") {
		t.Fatalf("expected whitebox request output at %s, got:\n%s", whiteboxPath, contents[whiteboxPath])
	}
}

func TestGenerateZodSchemas(t *testing.T) {
	tmp, err := os.CreateTemp("", "binding_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString(`package dto

import "encoding/json"

type UpdateProviderRoleMappingRequest struct {
	ExternalGroup *string ` + "`json:\"external_group\" binding:\"omitempty,min=1,max=512\"`" + `
	RoleID        *int64  ` + "`json:\"role_id\" binding:\"omitempty,gt=0\"`" + `
	Config        json.RawMessage ` + "`json:\"config\" binding:\"required\"`" + `
	Inputs        map[string]json.RawMessage ` + "`json:\"inputs\" binding:\"required\"`" + `
}

type CreateUserRequest struct {
	PrimaryEmail string  ` + "`json:\"primary_email\" binding:\"required,email\"`" + `
	DisplayName  string  ` + "`json:\"display_name\" binding:\"required,min=1,max=255\"`" + `
	Password     string  ` + "`json:\"password\" binding:\"required,min=8\"`" + `
	RoleIDs      []int64 ` + "`json:\"role_ids\" binding:\"required,min=1,dive,gt=0\"`" + `
	Status       *int16  ` + "`json:\"status\" binding:\"omitempty,oneof=1 4\"`" + `
}
`)
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	file, err := parser.ParseFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	g := NewGenerator(GeneratorConfig{ExportType: "interface"})
	outputs := g.GeneratePerFile([]*parser.File{file}, "/tmp/generated")

	var typeContent string
	var schemaContent string
	var utilsContent string
	for _, output := range outputs {
		switch output.Path {
		case filepath.Join("/tmp/generated", "utils.ts"):
			utilsContent = output.Content
		case filepath.Join("/tmp/generated", "unknown", file.Type+".ts"):
			typeContent = output.Content
		case filepath.Join("/tmp/generated", "unknown", file.Type+".schema.ts"):
			schemaContent = output.Content
		}
	}

	expectedTypeSnippets := []string{
		`import type { JsonValue } from '../utils';`,
		`config: JsonValue;`,
		`inputs: Record<string, JsonValue>;`,
		`export interface CreateUserRequest {`,
	}

	for _, snippet := range expectedTypeSnippets {
		if !strings.Contains(typeContent, snippet) {
			t.Fatalf("expected type snippet %q in output, got:\n%s", snippet, typeContent)
		}
	}

	unexpectedTypeSnippets := []string{
		`export type JsonValue =`,
		`import { z } from 'zod';`,
		`CreateUserRequestSchema`,
	}

	if !strings.Contains(utilsContent, jsonValueTypeDef) {
		t.Fatalf("expected shared JsonValue definition in utils.ts, got:\n%s", utilsContent)
	}
	for _, snippet := range unexpectedTypeSnippets {
		if strings.Contains(typeContent, snippet) {
			t.Fatalf("unexpected type snippet %q in output:\n%s", snippet, typeContent)
		}
	}

	expectedSchemaSnippets := []string{
		`import { z } from 'zod';`,
		`export const UpdateProviderRoleMappingRequestSchema = z.object({`,
		`"external_group": z.string().min(1).max(512).optional(),`,
		`"role_id": z.number().int().gt(0).optional(),`,
		`"config": z.json(),`,
		`export const CreateUserRequestSchema = z.object({`,
		`"primary_email": z.email(),`,
		`"display_name": z.string().min(1).max(255),`,
		`"password": z.string().min(8),`,
		`"role_ids": z.array(z.number().int().gt(0)).min(1),`,
		`"status": z.union([z.literal(1), z.literal(4)]).optional(),`,
	}

	for _, snippet := range expectedSchemaSnippets {
		if !strings.Contains(schemaContent, snippet) {
			t.Fatalf("expected schema snippet %q in output, got:\n%s", snippet, schemaContent)
		}
	}
}

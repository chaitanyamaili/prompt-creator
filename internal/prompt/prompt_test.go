package prompt

import (
	"strings"
	"testing"

	"prompt-creator/internal/category"
)

func testCategory() category.Category {
	return category.Category{
		Name: "test",
		Fields: []category.Field{
			{Key: "goal", Required: true},
			{Key: "context"},
			{Key: "constraints"},
		},
		Template: "Goal: {{.goal}}\n{{if .context}}\nContext:\n{{.context}}\n{{end}}{{if .constraints}}\nConstraints: {{.constraints}}\n{{end}}",
	}
}

func TestRenderIncludesOnlyAnsweredFields(t *testing.T) {
	out, err := Render(testCategory(), map[string]string{"goal": "Write a poem"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(out, "Goal: Write a poem") {
		t.Fatalf("expected goal line in output, got:\n%s", out)
	}
	if strings.Contains(out, "Context:") {
		t.Fatalf("expected blank optional field to be omitted, got:\n%s", out)
	}
	if strings.Contains(out, "Constraints:") {
		t.Fatalf("expected blank optional field to be omitted, got:\n%s", out)
	}
}

func TestRenderIncludesAllAnsweredFields(t *testing.T) {
	out, err := Render(testCategory(), map[string]string{
		"goal":        "Write a poem",
		"context":     "For a birthday card",
		"constraints": "Under 4 lines",
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	for _, want := range []string{"Goal: Write a poem", "Context:\nFor a birthday card", "Constraints: Under 4 lines"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRenderInvalidTemplateReturnsError(t *testing.T) {
	cat := testCategory()
	cat.Template = "{{.goal"
	_, err := Render(cat, map[string]string{"goal": "x"})
	if err == nil {
		t.Fatalf("expected error for invalid template")
	}
}

func TestRenderTemplateReferencingUnknownFieldKeyRendersBlank(t *testing.T) {
	cat := category.Category{
		Name: "test",
		Fields: []category.Field{
			{Key: "goal", Required: true},
		},
		// "extra" is not declared in Fields — a config typo.
		Template: "Goal: {{.goal}}\nExtra: {{.extra}}\n",
	}

	out, err := Render(cat, map[string]string{"goal": "Write a poem"})
	if err != nil {
		t.Fatalf("Render returned error for unknown template key: %v", err)
	}
	if !strings.Contains(out, "Goal: Write a poem") {
		t.Fatalf("expected goal line, got:\n%s", out)
	}
	if !strings.Contains(out, "Extra: \n") {
		t.Fatalf("expected unknown field to render blank rather than erroring, got:\n%s", out)
	}
}

func TestRenderRealGenericCategoryOmitsBlankOptionalFields(t *testing.T) {
	cats, err := category.Load(t.TempDir())
	if err != nil {
		t.Fatalf("category.Load returned error: %v", err)
	}
	generic, ok := cats["generic"]
	if !ok {
		t.Fatalf("expected built-in generic category")
	}

	out, err := Render(generic, map[string]string{"goal": "Refactor the parser"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if strings.TrimSpace(out) != "Goal: Refactor the parser" {
		t.Fatalf("expected only the goal line for an otherwise-blank generic prompt, got:\n%q", out)
	}
}

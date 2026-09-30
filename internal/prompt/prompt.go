package prompt

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/chaitanyamaili/prompt-creator/internal/category"
)

// Render executes the category's template against the provided field
// answers (keyed by Field.Key) and returns the assembled prompt text.
func Render(cat category.Category, answers map[string]string) (string, error) {
	tmpl, err := template.New(cat.Name).Option("missingkey=zero").Parse(cat.Template)
	if err != nil {
		return "", fmt.Errorf("parsing template for category %q: %w", cat.Name, err)
	}

	data := map[string]string{}
	for _, field := range cat.Fields {
		data[field.Key] = answers[field.Key]
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering template for category %q: %w", cat.Name, err)
	}
	return buf.String(), nil
}

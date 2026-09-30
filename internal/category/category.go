package category

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chaitanyamaili/prompt-creator/categories"
)

// Field describes one question the wizard/flag interface asks for a
// category.
type Field struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Question string `json:"question"`
	Required bool   `json:"required"`
}

// Category is a named field set plus the text/template used to assemble
// the final prompt from answers.
type Category struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
	Template    string  `json:"template"`
}

// Load returns all built-in categories overlaid with any user-defined
// categories found in userDir (a user category with the same Name replaces
// the built-in one of that name). A missing userDir is not an error.
// Non-.json entries and subdirectories in userDir are ignored.
func Load(userDir string) (map[string]Category, error) {
	result := map[string]Category{}

	builtinEntries, err := categories.FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded categories: %w", err)
	}
	for _, entry := range builtinEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := categories.FS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading embedded category %q: %w", entry.Name(), err)
		}
		cat, err := parse(entry.Name(), data)
		if err != nil {
			return nil, err
		}
		result[cat.Name] = cat
	}

	userEntries, err := os.ReadDir(userDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("reading user category dir %q: %w", userDir, err)
	}
	for _, entry := range userEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(userDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading user category %q: %w", path, err)
		}
		cat, err := parse(path, data)
		if err != nil {
			return nil, err
		}
		result[cat.Name] = cat
	}

	return result, nil
}

func parse(source string, data []byte) (Category, error) {
	var cat Category
	if err := json.Unmarshal(data, &cat); err != nil {
		return Category{}, fmt.Errorf("parsing category %q: %w", source, err)
	}
	if cat.Name == "" {
		return Category{}, fmt.Errorf("category %q: missing required \"name\" field", source)
	}
	return cat, nil
}

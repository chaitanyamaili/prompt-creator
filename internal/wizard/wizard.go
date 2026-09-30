package wizard

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/chaitanyamaili/prompt-creator/internal/category"
)

// Run asks the user each field's question for the given category, writing
// prompts to w and reading answers from scanner. Required fields are
// re-asked until a non-empty (after trimming whitespace) answer is given.
// It returns the collected answers keyed by Field.Key.
func Run(w io.Writer, scanner *bufio.Scanner, cat category.Category) (map[string]string, error) {
	answers := map[string]string{}

	for _, field := range cat.Fields {
		for {
			hint := " (optional, press Enter to skip)"
			if field.Required {
				hint = ""
			}
			fmt.Fprintf(w, "%s%s: ", field.Question, hint)

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("reading answer for %q: %w", field.Key, err)
				}
				return nil, fmt.Errorf("reading answer for %q: unexpected end of input", field.Key)
			}
			answer := strings.TrimSpace(scanner.Text())

			if answer == "" && field.Required {
				fmt.Fprintf(w, "%q is required.\n", field.Label)
				continue
			}

			answers[field.Key] = answer
			break
		}
	}

	return answers, nil
}

// Confirm asks a yes/no question, writing it to w and reading the answer
// from scanner. It returns true only for "y" or "yes" (case-insensitive).
func Confirm(w io.Writer, scanner *bufio.Scanner, question string) (bool, error) {
	fmt.Fprintf(w, "%s (y/n): ", question)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("reading confirmation: %w", err)
		}
		return false, fmt.Errorf("reading confirmation: unexpected end of input")
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

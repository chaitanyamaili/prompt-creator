package wizard

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"prompt-creator/internal/category"
)

func testCategory() category.Category {
	return category.Category{
		Name: "test",
		Fields: []category.Field{
			{Key: "goal", Label: "Goal", Question: "What do you want?", Required: true},
			{Key: "context", Label: "Context", Question: "Any background?", Required: false},
		},
	}
}

func TestRunCollectsAnswersInOrder(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"Write a poem", "For a birthday"}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["goal"] != "Write a poem" {
		t.Fatalf("expected goal answer, got %q", answers["goal"])
	}
	if answers["context"] != "For a birthday" {
		t.Fatalf("expected context answer, got %q", answers["context"])
	}
	if !strings.Contains(output.String(), "What do you want?") {
		t.Fatalf("expected required question without optional hint, got:\n%s", output.String())
	}
	if !strings.Contains(output.String(), "Any background? (optional, press Enter to skip)") {
		t.Fatalf("expected optional question with hint, got:\n%s", output.String())
	}
}

func TestRunSkipsOptionalFieldOnEmptyInput(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"Write a poem", ""}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["context"] != "" {
		t.Fatalf("expected empty context answer, got %q", answers["context"])
	}
}

func TestRunRepromptsOnEmptyRequiredField(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"", "", "Write a poem", ""}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["goal"] != "Write a poem" {
		t.Fatalf("expected goal answer after reprompt, got %q", answers["goal"])
	}
	if !strings.Contains(output.String(), `"Goal" is required.`) {
		t.Fatalf("expected reprompt message, got:\n%s", output.String())
	}
}

func TestRunRepromptsOnWhitespaceOnlyRequiredField(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"   ", "Write a poem", ""}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["goal"] != "Write a poem" {
		t.Fatalf("expected whitespace-only answer to be rejected and re-prompted, got %q", answers["goal"])
	}
}

func TestRunReturnsErrorOnUnexpectedEOF(t *testing.T) {
	input := strings.NewReader("")
	var output bytes.Buffer

	_, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err == nil {
		t.Fatalf("expected error when input ends before required field answered")
	}
}

func TestConfirmYes(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("y\n"))
	var output bytes.Buffer

	ok, err := Confirm(&output, scanner, "Save this?")
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true for 'y' answer")
	}
	if !strings.Contains(output.String(), "Save this? (y/n): ") {
		t.Fatalf("expected question in output, got: %s", output.String())
	}
}

func TestConfirmNo(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("n\n"))
	var output bytes.Buffer

	ok, err := Confirm(&output, scanner, "Save this?")
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected false for 'n' answer")
	}
}

func TestConfirmEmptyAnswerIsNo(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	var output bytes.Buffer

	ok, err := Confirm(&output, scanner, "Save this?")
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected false for empty answer")
	}
}

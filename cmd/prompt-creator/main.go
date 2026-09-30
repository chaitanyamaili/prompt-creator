package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"prompt-creator/internal/appdir"
	"prompt-creator/internal/category"
	"prompt-creator/internal/clipboard"
	"prompt-creator/internal/history"
	"prompt-creator/internal/prompt"
	"prompt-creator/internal/wizard"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runWizard(stdin, stdout, stderr)
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func runWizard(stdin io.Reader, stdout, stderr io.Writer) int {
	userDir, err := appdir.ConfigDir()
	if err != nil {
		fmt.Fprintf(stderr, "resolving config dir: %v\n", err)
		return 1
	}

	cats, err := category.Load(filepath.Join(userDir, "categories"))
	if err != nil {
		fmt.Fprintf(stderr, "loading categories: %v\n", err)
		return 1
	}

	cat, ok := cats["generic"]
	if !ok {
		fmt.Fprintln(stderr, "built-in \"generic\" category is missing")
		return 1
	}

	scanner := bufio.NewScanner(stdin)

	answers, err := wizard.Run(stdout, scanner, cat)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	rendered, err := prompt.Render(cat, answers)
	if err != nil {
		fmt.Fprintf(stderr, "rendering prompt: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, rendered)

	if err := clipboard.Copy(rendered); err != nil {
		fmt.Fprintf(stderr, "warning: %v\n", err)
	}

	save, err := wizard.Confirm(stdout, scanner, "Save this prompt to history?")
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if save {
		path, err := history.Save(history.Entry{
			Category: "generic",
			Answers:  answers,
			Prompt:   rendered,
		})
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not save history: %v\n", err)
		} else {
			fmt.Fprintf(stdout, "saved to %s\n", path)
		}
	}

	return 0
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	goal := fs.String("goal", "", "what you want the LLM to do (required)")
	contextAnswer := fs.String("context", "", "relevant background")
	constraints := fs.String("constraints", "", "format, length, tone, things to avoid")
	outputFormat := fs.String("output-format", "", "how the response should be structured")
	verbosity := fs.String("verbosity", "", "how much explanation the LLM should include")
	scopeBoundary := fs.String("scope-boundary", "", "anything explicitly out of scope")
	assumedKnowledge := fs.String("assumed-knowledge", "", "what the LLM should skip re-explaining")
	categoryName := fs.String("category", "generic", "category to use")
	save := fs.Bool("save", false, "save the generated prompt to history")
	noClipboard := fs.Bool("no-clipboard", false, "skip copying the prompt to the clipboard")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *goal == "" {
		fmt.Fprintln(stderr, "--goal is required")
		return 2
	}

	userDir, err := appdir.ConfigDir()
	if err != nil {
		fmt.Fprintf(stderr, "resolving config dir: %v\n", err)
		return 1
	}

	cats, err := category.Load(filepath.Join(userDir, "categories"))
	if err != nil {
		fmt.Fprintf(stderr, "loading categories: %v\n", err)
		return 1
	}

	cat, ok := cats[*categoryName]
	if !ok {
		fmt.Fprintf(stderr, "unknown category %q\n", *categoryName)
		return 2
	}

	answers := map[string]string{
		"goal":              *goal,
		"context":           *contextAnswer,
		"constraints":       *constraints,
		"output_format":     *outputFormat,
		"verbosity":         *verbosity,
		"scope_boundary":    *scopeBoundary,
		"assumed_knowledge": *assumedKnowledge,
	}

	rendered, err := prompt.Render(cat, answers)
	if err != nil {
		fmt.Fprintf(stderr, "rendering prompt: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, rendered)

	if !*noClipboard {
		if err := clipboard.Copy(rendered); err != nil {
			fmt.Fprintf(stderr, "warning: %v\n", err)
		}
	}

	if *save {
		path, err := history.Save(history.Entry{
			Category: *categoryName,
			Answers:  answers,
			Prompt:   rendered,
		})
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not save history: %v\n", err)
		} else {
			fmt.Fprintf(stderr, "saved to %s\n", path)
		}
	}

	return 0
}

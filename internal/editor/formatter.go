package editor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// formatDocument formats content for the given language/file path.
// Returns the formatted string or an error if formatting fails or is unsupported.
func formatDocument(content, language, filePath string) (string, error) {
	switch language {
	case "json":
		return formatJSON(content)
	case "html":
		return formatViaPrettier(content, filePath, "html")
	case "css", "scss", "less":
		return formatViaPrettier(content, filePath, "css")
	case "javascript":
		return formatViaPrettier(content, filePath, "babel")
	case "typescript":
		return formatViaPrettier(content, filePath, "typescript")
	case "sql":
		return formatSQL(content)
	default:
		return "", fmt.Errorf("no formatter available for '%s' — change language mode first if needed", language)
	}
}

func formatJSON(content string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(content), &v); err != nil {
		return "", fmt.Errorf("JSON parse error: %w", err)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out) + "\n", nil
}

// formatViaStdin runs an external formatter that reads from stdin and writes to stdout.
func formatViaStdin(content, name string, args ...string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("'%s' not found — install it to format this file", name)
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = strings.NewReader(content)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s: %s", name, msg)
	}
	return out.String(), nil
}

// formatViaPrettier runs prettier with --stdin-filepath for config-aware formatting.
func formatViaPrettier(content, filePath, parser string) (string, error) {
	prettierPath, err := exec.LookPath("prettier")
	if err != nil {
		return "", fmt.Errorf("'prettier' not found — install with: npm install -g prettier")
	}
	args := []string{"--stdin-filepath", filePath}
	if filePath == "" {
		args = []string{"--parser", parser}
	}
	cmd := exec.Command(prettierPath, args...)
	cmd.Stdin = strings.NewReader(content)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("prettier: %s", msg)
	}
	return out.String(), nil
}

// formatSQL tries sqlformat (Python) then sql-formatter (Node).
func formatSQL(content string) (string, error) {
	if path, err := exec.LookPath("sqlformat"); err == nil {
		out, err := runWithStdin(path, content, "--reindent", "--keywords", "upper", "-")
		if err == nil {
			return out, nil
		}
	}
	if path, err := exec.LookPath("sql-formatter"); err == nil {
		out, err := runWithStdin(path, content)
		if err == nil {
			return out, nil
		}
	}
	return "", fmt.Errorf("no SQL formatter found — install: pip install sqlparse  or  npm install -g sql-formatter")
}

func runWithStdin(path, input string, args ...string) (string, error) {
	cmd := exec.Command(path, args...)
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

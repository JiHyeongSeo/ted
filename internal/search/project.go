package search

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FileMatch represents a match in a specific file.
type FileMatch struct {
	File    string
	Line    int // 1-based line number
	Col     int // 1-based column
	Text    string
	MatchLen int
}

// ProjectSearch searches across files in a project directory.
type ProjectSearch struct {
	root     string
	excludes []string
	useRg    bool
}

// NewProjectSearch creates a new project search rooted at the given directory.
func NewProjectSearch(root string, excludes []string, useRg bool) *ProjectSearch {
	return &ProjectSearch{
		root:     root,
		excludes: excludes,
		useRg:    useRg,
	}
}

// Search performs a project-wide search for the given pattern.
func (ps *ProjectSearch) Search(pattern string, useRegex bool) ([]FileMatch, error) {
	if ps.useRg {
		if _, err := exec.LookPath("rg"); err == nil {
			return ps.searchWithRipgrep(pattern, useRegex)
		}
	}
	return ps.searchWithGo(pattern, useRegex)
}

func (ps *ProjectSearch) searchWithRipgrep(pattern string, useRegex bool) ([]FileMatch, error) {
	args := []string{"--line-number", "--column", "--no-heading"}
	if !useRegex {
		args = append(args, "--fixed-strings")
	}
	for _, exc := range ps.excludes {
		args = append(args, "--glob", "!"+exc)
	}
	args = append(args, pattern, ps.root)

	cmd := exec.Command("rg", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil // no matches
		}
		return nil, err
	}

	return parseRipgrepOutput(string(output)), nil
}

func parseRipgrepOutput(output string) []FileMatch {
	var matches []FileMatch
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: file:line:col:text
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		lineNum, _ := strconv.Atoi(parts[1])
		col, _ := strconv.Atoi(parts[2])
		matches = append(matches, FileMatch{
			File: parts[0],
			Line: lineNum,
			Col:  col,
			Text: parts[3],
		})
	}
	return matches
}

func (ps *ProjectSearch) searchWithGo(pattern string, useRegex bool) ([]FileMatch, error) {
	var re *regexp.Regexp
	var err error

	if useRegex {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}

	var matches []FileMatch

	err = filepath.Walk(ps.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			base := info.Name()
			for _, exc := range ps.excludes {
				if base == exc {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Skip binary files (simple heuristic: check first 512 bytes)
		if info.Size() > 10*1024*1024 { // skip files > 10MB
			return nil
		}

		fileMatches, err := ps.searchFile(path, pattern, useRegex, re)
		if err != nil {
			return nil
		}
		matches = append(matches, fileMatches...)
		return nil
	})

	return matches, err
}

func (ps *ProjectSearch) searchFile(path string, pattern string, useRegex bool, re *regexp.Regexp) ([]FileMatch, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var matches []FileMatch
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if useRegex && re != nil {
			loc := re.FindStringIndex(line)
			if loc != nil {
				matches = append(matches, FileMatch{
					File:     path,
					Line:     lineNum,
					Col:      loc[0] + 1,
					Text:     line,
					MatchLen: loc[1] - loc[0],
				})
			}
		} else {
			idx := strings.Index(line, pattern)
			if idx >= 0 {
				matches = append(matches, FileMatch{
					File:     path,
					Line:     lineNum,
					Col:      idx + 1,
					Text:     line,
					MatchLen: len(pattern),
				})
			}
		}
	}

	return matches, scanner.Err()
}

// SearchResult formats a FileMatch for display.
func (fm FileMatch) String() string {
	return fmt.Sprintf("%s:%d:%d: %s", fm.File, fm.Line, fm.Col, fm.Text)
}

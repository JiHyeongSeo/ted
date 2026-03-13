package search

import (
	"regexp"
	"strings"
)

// Match represents a single search match within text.
type Match struct {
	Line      int // 0-based line number
	Col       int // 0-based column (byte offset within line)
	Length    int // match length in bytes
	LineText  string
}

// InFileSearch performs text searching within a buffer's content.
type InFileSearch struct {
	query    string
	regex    *regexp.Regexp
	useRegex bool
	caseSensitive bool
}

// NewInFileSearch creates a new search with the given query.
func NewInFileSearch(query string, useRegex bool, caseSensitive bool) (*InFileSearch, error) {
	s := &InFileSearch{
		query:         query,
		useRegex:      useRegex,
		caseSensitive: caseSensitive,
	}

	if useRegex {
		pattern := query
		if !caseSensitive {
			pattern = "(?i)" + pattern
		}
		var err error
		s.regex, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

// FindAll returns all matches in the given text.
func (s *InFileSearch) FindAll(text string) []Match {
	lines := strings.Split(text, "\n")
	var matches []Match

	for lineNum, line := range lines {
		lineMatches := s.findInLine(line, lineNum)
		matches = append(matches, lineMatches...)
	}

	return matches
}

// FindNext returns the next match starting from the given line and column.
// Returns nil if no match is found.
func (s *InFileSearch) FindNext(text string, fromLine, fromCol int) *Match {
	lines := strings.Split(text, "\n")

	// Search from current position to end
	for lineNum := fromLine; lineNum < len(lines); lineNum++ {
		line := lines[lineNum]
		startCol := 0
		if lineNum == fromLine {
			startCol = fromCol
		}
		if m := s.findFirstInLine(line, lineNum, startCol); m != nil {
			return m
		}
	}

	// Wrap around: search from beginning to current position
	for lineNum := 0; lineNum <= fromLine && lineNum < len(lines); lineNum++ {
		line := lines[lineNum]
		endCol := len(line)
		if lineNum == fromLine {
			endCol = fromCol
		}
		if m := s.findFirstInLine(line, lineNum, 0); m != nil {
			if m.Col < endCol {
				return m
			}
		}
	}

	return nil
}

func (s *InFileSearch) findInLine(line string, lineNum int) []Match {
	var matches []Match

	if s.useRegex && s.regex != nil {
		locs := s.regex.FindAllStringIndex(line, -1)
		for _, loc := range locs {
			matches = append(matches, Match{
				Line:     lineNum,
				Col:      loc[0],
				Length:   loc[1] - loc[0],
				LineText: line,
			})
		}
	} else {
		searchLine := line
		searchQuery := s.query
		if !s.caseSensitive {
			searchLine = strings.ToLower(line)
			searchQuery = strings.ToLower(s.query)
		}

		offset := 0
		for {
			idx := strings.Index(searchLine[offset:], searchQuery)
			if idx < 0 {
				break
			}
			matches = append(matches, Match{
				Line:     lineNum,
				Col:      offset + idx,
				Length:   len(s.query),
				LineText: line,
			})
			offset += idx + len(searchQuery)
		}
	}

	return matches
}

func (s *InFileSearch) findFirstInLine(line string, lineNum int, startCol int) *Match {
	if startCol >= len(line) {
		return nil
	}

	if s.useRegex && s.regex != nil {
		loc := s.regex.FindStringIndex(line[startCol:])
		if loc != nil {
			return &Match{
				Line:     lineNum,
				Col:      startCol + loc[0],
				Length:   loc[1] - loc[0],
				LineText: line,
			}
		}
	} else {
		searchLine := line[startCol:]
		searchQuery := s.query
		if !s.caseSensitive {
			searchLine = strings.ToLower(searchLine)
			searchQuery = strings.ToLower(s.query)
		}
		idx := strings.Index(searchLine, searchQuery)
		if idx >= 0 {
			return &Match{
				Line:     lineNum,
				Col:      startCol + idx,
				Length:   len(s.query),
				LineText: line,
			}
		}
	}

	return nil
}

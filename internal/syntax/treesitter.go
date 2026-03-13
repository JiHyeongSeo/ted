package syntax

import (
	"context"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/python"
)

// TSHighlighter provides tree-sitter based syntax highlighting.
type TSHighlighter struct {
	theme    *Theme
	language string
	parser   *sitter.Parser
	tree     *sitter.Tree
	source   []byte // source used to build current tree
	mu       sync.Mutex
}

// NewTSHighlighter creates a tree-sitter highlighter for the given language.
func NewTSHighlighter(theme *Theme, language string) *TSHighlighter {
	parser := sitter.NewParser()

	var lang *sitter.Language
	switch language {
	case "go":
		lang = golang.GetLanguage()
	case "python":
		lang = python.GetLanguage()
	default:
		return nil
	}

	parser.SetLanguage(lang)

	return &TSHighlighter{
		theme:    theme,
		language: language,
		parser:   parser,
	}
}

// Parse parses the full source code and stores the tree.
func (h *TSHighlighter) Parse(source []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Keep a copy of the source so HighlightLine uses the exact same bytes
	src := make([]byte, len(source))
	copy(src, source)

	// Pass nil as old tree to force full reparse.
	// Incremental parsing requires tree.Edit() calls which we don't track.
	tree, err := h.parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return
	}
	h.tree = tree
	h.source = src
}

// HighlightLine returns tokens for a specific line using the stored AST and source.
func (h *TSHighlighter) HighlightLine(lineNum int) []Token {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.tree == nil || h.source == nil {
		return nil
	}

	root := h.tree.RootNode()

	// Find the byte range for this line using the same source that was parsed
	lineStart, lineEnd := lineByteRange(h.source, lineNum)
	if lineStart >= lineEnd {
		return nil
	}

	var tokens []Token
	h.walkNode(root, lineNum, lineStart, lineEnd, h.source, &tokens)

	return tokens
}

func (h *TSHighlighter) walkNode(node *sitter.Node, lineNum int, lineStart, lineEnd uint32, source []byte, tokens *[]Token) {
	startByte := node.StartByte()
	endByte := node.EndByte()

	// Skip nodes entirely outside our line
	if endByte <= lineStart || startByte >= lineEnd {
		return
	}

	childCount := int(node.ChildCount())

	// If this is a leaf node (or a node we want to highlight), map it
	if childCount == 0 {
		tt := nodeToTokenType(node, h.language)
		if tt != TokenNone {
			// Clamp to line boundaries
			s := startByte
			e := endByte
			if s < lineStart {
				s = lineStart
			}
			if e > lineEnd {
				e = lineEnd
			}
			// Convert byte offsets to rune-based offsets within the line
			lineContent := source[lineStart:lineEnd]
			runeStart := byteOffsetToRuneOffset(lineContent, int(s-lineStart))
			runeLen := byteOffsetToRuneOffset(lineContent[s-lineStart:], int(e-s))

			*tokens = append(*tokens, Token{
				Start:  runeStart,
				Length: runeLen,
				Type:   tt,
			})
		}
		return
	}

	// Recurse into children
	for i := 0; i < childCount; i++ {
		child := node.Child(i)
		h.walkNode(child, lineNum, lineStart, lineEnd, source, tokens)
	}
}

// nodeToTokenType maps a tree-sitter AST node to a token type.
func nodeToTokenType(node *sitter.Node, language string) TokenType {
	nodeType := node.Type()
	parentType := ""
	if node.Parent() != nil {
		parentType = node.Parent().Type()
	}

	// Common across languages
	switch nodeType {
	case "comment", "line_comment", "block_comment":
		return TokenComment
	case "interpreted_string_literal", "raw_string_literal",
		"string", "string_literal", "concatenated_string",
		"template_string":
		return TokenString
	case "int_literal", "float_literal", "integer", "float",
		"imaginary_literal", "none", "true", "false":
		return TokenNumber
	case "nil_type":
		return TokenConstant
	}

	switch language {
	case "go":
		return goNodeToToken(nodeType, parentType)
	case "python":
		return pythonNodeToToken(nodeType, parentType)
	}

	return TokenNone
}

func goNodeToToken(nodeType, parentType string) TokenType {
	switch nodeType {
	// Keywords
	case "break", "case", "chan", "const", "continue", "default",
		"defer", "else", "fallthrough", "for", "func", "go", "goto",
		"if", "import", "interface", "map", "package", "range", "return",
		"select", "struct", "switch", "type", "var":
		return TokenKeyword

	// Built-in types
	case "type_identifier":
		return TokenType_
	case "field_identifier":
		return TokenProperty
	case "package_identifier":
		return TokenVariable

	// Function calls
	case "identifier":
		if parentType == "call_expression" || parentType == "function_declaration" {
			return TokenFunction
		}
		return TokenVariable

	// Operators
	case "+", "-", "*", "/", "%", "=", "==", "!=", "<", ">",
		"<=", ">=", "&&", "||", "!", "&", "|", "^", "<<", ">>",
		":=", "+=", "-=", "*=", "/=", "<-":
		return TokenOperator

	// Punctuation
	case "(", ")", "{", "}", "[", "]", ",", ".", ";", ":":
		return TokenPunctuation
	}

	return TokenNone
}

func pythonNodeToToken(nodeType, parentType string) TokenType {
	switch nodeType {
	// Keywords
	case "and", "as", "assert", "async", "await", "break",
		"class", "continue", "def", "del", "elif", "else", "except",
		"finally", "for", "from", "global", "if", "import", "in", "is",
		"lambda", "nonlocal", "not", "or", "pass", "raise", "return",
		"try", "while", "with", "yield":
		return TokenKeyword

	case "identifier":
		if parentType == "call" || parentType == "function_definition" ||
			parentType == "decorated_definition" {
			return TokenFunction
		}
		if parentType == "class_definition" {
			return TokenType_
		}
		return TokenVariable

	case "type":
		return TokenType_

	case "attribute":
		return TokenProperty

	// String types
	case "string_content", "escape_sequence":
		return TokenString

	// Operators
	case "+", "-", "*", "/", "//", "%", "**", "=", "==", "!=",
		"<", ">", "<=", ">=",
		"+=", "-=", "*=", "/=", "//=", "%=", "**=", "|", "&", "^",
		"~", "<<", ">>", "@":
		return TokenOperator

	// Decorators
	case "decorator":
		return TokenKeyword

	// Punctuation
	case "(", ")", "{", "}", "[", "]", ",", ".", ";", ":":
		return TokenPunctuation
	}

	return TokenNone
}

// lineByteRange returns the start and end byte offsets for a given line number.
func lineByteRange(source []byte, lineNum int) (uint32, uint32) {
	line := 0
	start := 0
	for i, b := range source {
		if line == lineNum {
			start = i
			break
		}
		if b == '\n' {
			line++
		}
	}
	if line < lineNum {
		return 0, 0
	}

	// Find end of line
	end := len(source)
	for i := start; i < len(source); i++ {
		if source[i] == '\n' {
			end = i
			break
		}
	}

	return uint32(start), uint32(end)
}

// byteOffsetToRuneOffset converts a byte offset to a rune offset within a slice.
func byteOffsetToRuneOffset(data []byte, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}
	if byteOff > len(data) {
		byteOff = len(data)
	}
	return len([]rune(string(data[:byteOff])))
}

// Supported returns whether tree-sitter highlighting is available for a language.
func TSSupported(language string) bool {
	switch language {
	case "go", "python":
		return true
	}
	return false
}

// Language returns the language of this highlighter.
func (h *TSHighlighter) Language() string {
	return h.language
}

// nodeTypeIsKeyword checks if a node type string matches common keywords.
func nodeTypeIsKeyword(nodeType string) bool {
	return strings.HasSuffix(nodeType, "_keyword") || strings.HasSuffix(nodeType, "_statement")
}

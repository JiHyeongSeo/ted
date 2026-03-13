package syntax

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// TokenType represents a syntax token category.
type TokenType string

const (
	TokenKeyword     TokenType = "keyword"
	TokenString      TokenType = "string"
	TokenComment     TokenType = "comment"
	TokenFunction    TokenType = "function"
	TokenType_       TokenType = "type"
	TokenNumber      TokenType = "number"
	TokenOperator    TokenType = "operator"
	TokenVariable    TokenType = "variable"
	TokenConstant    TokenType = "constant"
	TokenProperty    TokenType = "property"
	TokenPunctuation TokenType = "punctuation"
	TokenNone        TokenType = ""
)

// Token represents a styled span within a line.
type Token struct {
	Start  int       // byte offset within the line
	Length int       // byte length
	Type   TokenType // token category for styling
}

// Highlighter provides syntax highlighting for buffer content.
// This is a simple keyword-based highlighter. A tree-sitter based
// implementation can replace this for more accurate highlighting.
type Highlighter struct {
	theme    *Theme
	language string
	keywords map[string]bool
	types    map[string]bool
	builtins map[string]bool
}

// NewHighlighter creates a highlighter for the given language.
func NewHighlighter(theme *Theme, language string) *Highlighter {
	h := &Highlighter{
		theme:    theme,
		language: language,
	}
	h.loadLanguage(language)
	return h
}

// HighlightLine returns tokens for a single line of text.
func (h *Highlighter) HighlightLine(line string) []Token {
	if h.language == "text" || h.language == "" {
		return nil
	}

	var tokens []Token
	i := 0

	for i < len(line) {
		// Skip whitespace
		if line[i] == ' ' || line[i] == '\t' {
			i++
			continue
		}

		// String literals
		if line[i] == '"' || line[i] == '\'' || line[i] == '`' {
			quote := line[i]
			start := i
			i++
			for i < len(line) && line[i] != quote {
				if line[i] == '\\' {
					i++ // skip escape
				}
				i++
			}
			if i < len(line) {
				i++ // closing quote
			}
			tokens = append(tokens, Token{Start: start, Length: i - start, Type: TokenString})
			continue
		}

		// Line comments
		if i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
			tokens = append(tokens, Token{Start: i, Length: len(line) - i, Type: TokenComment})
			break
		}
		if line[i] == '#' && (h.language == "python" || h.language == "bash" || h.language == "ruby" || h.language == "yaml") {
			tokens = append(tokens, Token{Start: i, Length: len(line) - i, Type: TokenComment})
			break
		}

		// Numbers
		if line[i] >= '0' && line[i] <= '9' {
			start := i
			for i < len(line) && (line[i] >= '0' && line[i] <= '9' || line[i] == '.' || line[i] == 'x' || line[i] == 'X' || (line[i] >= 'a' && line[i] <= 'f') || (line[i] >= 'A' && line[i] <= 'F')) {
				i++
			}
			tokens = append(tokens, Token{Start: start, Length: i - start, Type: TokenNumber})
			continue
		}

		// Identifiers and keywords
		if isIdentStart(line[i]) {
			start := i
			for i < len(line) && isIdentChar(line[i]) {
				i++
			}
			word := line[start:i]

			var tt TokenType
			if h.keywords[word] {
				tt = TokenKeyword
			} else if h.types[word] {
				tt = TokenType_
			} else if h.builtins[word] {
				tt = TokenFunction
			} else if i < len(line) && line[i] == '(' {
				tt = TokenFunction
			} else {
				tt = TokenVariable
			}
			tokens = append(tokens, Token{Start: start, Length: i - start, Type: tt})
			continue
		}

		// Operators and punctuation
		if strings.ContainsRune("+-*/%=<>!&|^~?:", rune(line[i])) {
			tokens = append(tokens, Token{Start: i, Length: 1, Type: TokenOperator})
			i++
			continue
		}
		if strings.ContainsRune("(){}[].,;", rune(line[i])) {
			tokens = append(tokens, Token{Start: i, Length: 1, Type: TokenPunctuation})
			i++
			continue
		}

		i++
	}

	return tokens
}

// StyleForToken returns the tcell.Style for a token type.
func (h *Highlighter) StyleForToken(tt TokenType) tcell.Style {
	if h.theme == nil {
		return tcell.StyleDefault
	}
	return h.theme.TokenStyle(string(tt))
}

func (h *Highlighter) loadLanguage(lang string) {
	switch lang {
	case "go":
		h.keywords = setOf("break", "case", "chan", "const", "continue", "default",
			"defer", "else", "fallthrough", "for", "func", "go", "goto", "if",
			"import", "interface", "map", "package", "range", "return", "select",
			"struct", "switch", "type", "var")
		h.types = setOf("bool", "byte", "complex64", "complex128", "error",
			"float32", "float64", "int", "int8", "int16", "int32", "int64",
			"rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr")
		h.builtins = setOf("append", "cap", "close", "copy", "delete", "len",
			"make", "new", "panic", "print", "println", "recover")
	case "python":
		h.keywords = setOf("and", "as", "assert", "async", "await", "break",
			"class", "continue", "def", "del", "elif", "else", "except",
			"finally", "for", "from", "global", "if", "import", "in", "is",
			"lambda", "nonlocal", "not", "or", "pass", "raise", "return",
			"try", "while", "with", "yield")
		h.types = setOf("int", "float", "str", "bool", "list", "dict", "tuple",
			"set", "None", "True", "False")
		h.builtins = setOf("print", "len", "range", "type", "isinstance",
			"enumerate", "zip", "map", "filter", "sorted", "reversed",
			"input", "open", "super", "property")
	case "javascript", "typescript":
		h.keywords = setOf("break", "case", "catch", "class", "const", "continue",
			"debugger", "default", "delete", "do", "else", "export", "extends",
			"finally", "for", "function", "if", "import", "in", "instanceof",
			"let", "new", "of", "return", "super", "switch", "this", "throw",
			"try", "typeof", "var", "void", "while", "with", "yield", "async", "await")
		h.types = setOf("boolean", "number", "string", "object", "symbol",
			"undefined", "null", "true", "false", "any", "void", "never")
		h.builtins = setOf("console", "Math", "JSON", "Array", "Object",
			"Promise", "setTimeout", "setInterval", "fetch", "require")
	case "rust":
		h.keywords = setOf("as", "break", "const", "continue", "crate", "else",
			"enum", "extern", "false", "fn", "for", "if", "impl", "in", "let",
			"loop", "match", "mod", "move", "mut", "pub", "ref", "return",
			"self", "static", "struct", "super", "trait", "true", "type",
			"unsafe", "use", "where", "while", "async", "await", "dyn")
		h.types = setOf("bool", "char", "f32", "f64", "i8", "i16", "i32", "i64",
			"i128", "isize", "str", "u8", "u16", "u32", "u64", "u128", "usize",
			"String", "Vec", "Option", "Result", "Box")
		h.builtins = setOf("println", "print", "eprintln", "eprint", "format",
			"vec", "panic", "todo", "unimplemented", "unreachable")
	default:
		h.keywords = make(map[string]bool)
		h.types = make(map[string]bool)
		h.builtins = make(map[string]bool)
	}
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isIdentChar(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

func setOf(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

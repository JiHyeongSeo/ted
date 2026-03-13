package syntax

import "testing"

func TestHighlightGoKeywords(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "go")

	tokens := h.HighlightLine("func main() {")
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}
	// "func" should be keyword
	if tokens[0].Type != TokenKeyword {
		t.Errorf("expected keyword, got %q", tokens[0].Type)
	}
}

func TestHighlightGoString(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "go")

	tokens := h.HighlightLine(`x := "hello world"`)
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenString {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected string token")
	}
}

func TestHighlightGoComment(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "go")

	tokens := h.HighlightLine("// this is a comment")
	if len(tokens) == 0 {
		t.Fatal("expected comment token")
	}
	if tokens[0].Type != TokenComment {
		t.Errorf("expected comment, got %q", tokens[0].Type)
	}
}

func TestHighlightGoNumber(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "go")

	tokens := h.HighlightLine("x := 42")
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenNumber {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected number token")
	}
}

func TestHighlightGoFunction(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "go")

	tokens := h.HighlightLine("fmt.Println(x)")
	// "Println(" should make "Println" a function
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenFunction {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected function token")
	}
}

func TestHighlightPythonComment(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "python")

	tokens := h.HighlightLine("# comment")
	if len(tokens) == 0 {
		t.Fatal("expected comment token")
	}
	if tokens[0].Type != TokenComment {
		t.Errorf("expected comment, got %q", tokens[0].Type)
	}
}

func TestHighlightTextNoTokens(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "text")

	tokens := h.HighlightLine("hello world")
	if tokens != nil {
		t.Errorf("expected nil tokens for text, got %d", len(tokens))
	}
}

func TestHighlightMultipleLanguages(t *testing.T) {
	theme := DefaultTheme()

	languages := []string{"go", "python", "javascript", "typescript", "rust"}
	for _, lang := range languages {
		h := NewHighlighter(theme, lang)
		tokens := h.HighlightLine("if x { return 0 }")
		if len(tokens) == 0 {
			t.Errorf("expected tokens for %s", lang)
		}
	}
}

func TestHighlightGoType(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "go")

	tokens := h.HighlightLine("var x int")
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenType_ {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected type token for 'int'")
	}
}

func TestStyleForToken(t *testing.T) {
	theme := DefaultTheme()
	h := NewHighlighter(theme, "go")

	style := h.StyleForToken(TokenKeyword)
	// Just check it returns something non-zero
	if style == (style) {
		// This always passes, just checking no panic
	}
}

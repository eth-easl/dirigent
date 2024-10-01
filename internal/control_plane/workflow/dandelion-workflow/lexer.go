package dandelion_workflow

import (
	"bufio"
	"bytes"
	"fmt"
)

type Token int

const (
	// special tokens
	ILLEGAL Token = iota
	EOF
	SPACE

	// identifiers
	IDENT

	// keywords
	FUNC             // :function
	COMP             // :composition
	ALL              // :all
	KEYED            // :keyed
	EACH             // :each
	LOOP             // :loop
	UNTIL_EMPTY      // :until_empty
	UNTIL_EMPTY_ITEM // :until_empty_item
	FEEDBACK         // :feedback

	// symbols
	BOPEN  // (
	BCLOSE // )
	FROM   // <-
	TOTHIN // ->
	TOFAT  // =>
	GETS   // :=
)

var eof = rune(0)

func is_space(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

func is_letter(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

type Lexer struct {
	in   *bufio.Reader
	curr rune
}

func NewLexer(r *bufio.Reader) *Lexer {
	return &Lexer{in: r}
}

func (l *Lexer) read() rune {
	c, _, err := l.in.ReadRune()
	if err != nil {
		return eof
	}
	return c
}

func (l *Lexer) unread() {
	_ = l.in.UnreadRune()
}

func (l *Lexer) scanSpace() (Token, string) {
	buf := bytes.Buffer{}
	buf.WriteRune(' ')

	for {
		c := l.read()

		if !is_space(c) {
			break
		}
	}

	l.unread()
	return SPACE, buf.String()
}

func (l *Lexer) scanIdent() (Token, string) {
	buf := bytes.Buffer{}
	buf.WriteRune(l.curr)

	for {
		c := l.read()

		if !is_letter(c) {
			break
		}
		buf.WriteRune(c)
	}

	l.unread()
	return IDENT, buf.String()
}

func (l *Lexer) Scan() (Token, string) {
	c := l.read()

	// comments
	for c == ';' {
		c = l.read()
		for c != '\n' {
			if c == eof {
				return EOF, ""
			}
			c = l.read()
		}
		c = l.read()
	}

	// whitespaces or identifiers
	if is_space(c) {
		l.curr = c
		return l.scanSpace()
	}
	if is_letter(c) {
		l.curr = c
		return l.scanIdent()
	}

	// keywords (or ":=")
	if c == ':' {
		c = l.read()

		// special case ":="
		if c == '=' {
			return GETS, ":="
		}

		// scan keyword
		l.curr = c
		_, str := l.scanIdent()

		switch str {
		case "function":
			return FUNC, fmt.Sprintf(":%s", str)
		case "composition":
			return COMP, fmt.Sprintf(":%s", str)
		case "all":
			return ALL, fmt.Sprintf(":%s", str)
		case "keyed":
			return KEYED, fmt.Sprintf(":%s", str)
		case "each":
			return EACH, fmt.Sprintf(":%s", str)
		case "loop":
			return LOOP, fmt.Sprintf(":%s", str)
		case "until_empty":
			return UNTIL_EMPTY, fmt.Sprintf(":%s", str)
		case "until_empty_item":
			return UNTIL_EMPTY_ITEM, fmt.Sprintf(":%s", str)
		case "feedback":
			return FEEDBACK, fmt.Sprintf(":%s", str)
		default:
			return ILLEGAL, fmt.Sprintf(":%s", str)
		}
	}

	// symbols
	if c == '(' {
		return BOPEN, "("
	}
	if c == ')' {
		return BCLOSE, ")"
	}
	if c == '-' {
		c = l.read()
		if c == '>' {
			return TOTHIN, "->"
		} else {
			return ILLEGAL, fmt.Sprintf("-%c", c)
		}
	}
	if c == '=' {
		c = l.read()
		if c == '>' {
			return TOFAT, "->"
		} else {
			return ILLEGAL, fmt.Sprintf("-%c", c)
		}
	}
	if c == '<' {
		c = l.read()
		if c == '-' {
			return FROM, "<-"
		} else {
			return ILLEGAL, fmt.Sprintf("<%c", c)
		}
	}

	if c == eof {
		return EOF, ""
	}

	return ILLEGAL, ""
}

func (l *Lexer) ScanIgnoreSpace() (Token, string) {
	for {
		t, s := l.Scan()
		if t != SPACE {
			return t, s
		}
	}
}

func (l *Lexer) SafeScanIgnoreSpace() (Token, string, error) {
	t, s := l.ScanIgnoreSpace()

	switch t {
	case ILLEGAL:
		return ILLEGAL, s, fmt.Errorf("lexer found illegal token: %s", s)
	case EOF:
		return EOF, s, fmt.Errorf("unexpectedly reached end of file")
	default:
		return t, s, nil
	}
}

func (l *Lexer) ExpectIgnoreSpace(expected Token, expectedDesc string) (Token, string, error) {
	t, s := l.ScanIgnoreSpace()

	switch t {
	case expected:
		return t, s, nil
	case ILLEGAL:
		return ILLEGAL, s, fmt.Errorf("lexer found illegal token: %s", s)
	case EOF:
		return EOF, s, fmt.Errorf("expected %s but reached end of file", expectedDesc)
	default:
		return t, s, fmt.Errorf("expected %s but got %s", expectedDesc, s)
	}
}

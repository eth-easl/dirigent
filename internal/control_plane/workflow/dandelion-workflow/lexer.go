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
	FUNC  // function
	COMP  // composition
	ALL   // all
	KEYED // keyed
	EACH  // each
	ANY   // any

	// symbols
	POPEN     // (
	PCLOSE    // )
	BOPEN     // {
	BCLOSE    // }
	TO        // =>
	EQUALS    // =
	COMMA     // ,
	SEMICOLON // ;
)

var eof = rune(0)

func is_space(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

func is_letter(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

type Lexer struct {
	in      *bufio.Reader
	curr    rune
	lineIdx int
	colIdx  int

	nextToken Token
	nextBuf   string
}

func NewLexer(r *bufio.Reader) *Lexer {
	return &Lexer{in: r, lineIdx: 0, colIdx: -1}
}

func (l *Lexer) PositionStr() string {
	return fmt.Sprintf("line=%d, col=%d", l.lineIdx, l.colIdx)
}

func (l *Lexer) read() rune {
	c, _, err := l.in.ReadRune()
	if err != nil {
		return eof
	}
	l.colIdx++
	if c == '\n' {
		l.lineIdx++
		l.colIdx = 0
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
	// if next token has been read already using Peek()
	if l.nextToken != ILLEGAL {
		t := l.nextToken
		b := l.nextBuf
		l.nextToken = ILLEGAL
		l.nextBuf = ""
		return t, b
	}

	c := l.read()

	// comments
	for c == '#' {
		c = l.read()
		for c != '\n' {
			if c == eof {
				return EOF, ""
			}
			c = l.read()
		}
		c = l.read()
	}

	// whitespace
	if is_space(c) {
		l.curr = c
		return l.scanSpace()
	}

	// identifiers or keywords
	if is_letter(c) {
		l.curr = c
		_, str := l.scanIdent()
		switch str {
		case "function":
			return FUNC, str
		case "composition":
			return COMP, str
		case "all":
			return ALL, str
		case "keyed":
			return KEYED, str
		case "each":
			return EACH, str
		case "any":
			return ANY, str
		default:
			return IDENT, str
		}
	}

	// symbols
	if c == '(' {
		return POPEN, "("
	}
	if c == ')' {
		return PCLOSE, ")"
	}
	if c == '{' {
		return BOPEN, "{"
	}
	if c == '}' {
		return BCLOSE, "}"
	}
	if c == '=' {
		c = l.read()
		if is_space(c) {
			l.unread()
			return EQUALS, "="
		} else if c == '>' {
			return TO, "=>"
		} else {
			return ILLEGAL, fmt.Sprintf("=%c", c)
		}
	}
	if c == ',' {
		return COMMA, ","
	}
	if c == ';' {
		return SEMICOLON, ";"
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
		return ILLEGAL, s, fmt.Errorf("lexer found illegal token '%s' at line=%d, col=%d", s, l.lineIdx, l.colIdx)
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
		return ILLEGAL, s, fmt.Errorf("lexer found illegal token '%s' at %s", s, l.PositionStr())
	case EOF:
		return EOF, s, fmt.Errorf("expected %s but reached end of file", expectedDesc)
	default:
		return t, s, fmt.Errorf("expected %s but got %s at %s", expectedDesc, s, l.PositionStr())
	}
}

func (l *Lexer) Peek() (Token, string) {
	if l.nextToken != ILLEGAL {
		return ILLEGAL, l.nextBuf
	}
	l.nextToken, l.nextBuf = l.Scan()
	return l.nextToken, l.nextBuf
}

func (l *Lexer) ConsumePeek() {
	l.nextToken = ILLEGAL
	l.nextBuf = ""
}

func (l *Lexer) PeekIgnoreSpace() (Token, string) {
	for {
		t, s := l.Peek()
		if t != SPACE {
			return t, s
		}
		l.ConsumePeek()
	}
}

func (l *Lexer) SafePeekIgnoreSpace() (Token, string, error) {
	t, s := l.PeekIgnoreSpace()

	switch t {
	case ILLEGAL:
		return ILLEGAL, s, fmt.Errorf("lexer found illegal token '%s' at %s", s, l.PositionStr())
	default:
		return t, s, nil
	}
}

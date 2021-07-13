package goql

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type Token int

func (t Token) String() string {
	switch t {
	case EOF:
		return "EOF"
	case WS:
		return "WS"
	case OP_EQI:
		return "OP_EQI"
	case OP_EQ:
		return "OP_EQ"
	case OP_NEQI:
		return "OP_NEQI"
	case OP_NEQ:
		return "OP_NEQ"
	case OP_RX:
		return "OP_RX"
	case OP_RXN:
		return "OP_RXN"
	case LNK_AND:
		return "LNK_AND"
	case LNK_OR:
		return "LNK_OR"
	case OP_GT:
		return "OP_GT"
	case OP_GE:
		return "OP_GE"
	case OP_LT:
		return "OP_LT"
	case OP_LE:
		return "OP_LE"
	case LITERAL:
		return "LITERAL"
	case INTEGER:
		return "INTEGER"
	case FLOAT:
		return "FLOAT"
	case BOOLEAN:
		return "BOOLEAN"
	case TIME:
		return "TIME"
	case ILLEGAL:
		return "ILLEGAL"
	default:
		return "UNKNOWN"
	}
}

func (t Token) isLiteral() bool {
	return t == LITERAL || t == INTEGER || t == FLOAT || t == BOOLEAN || t == TIME
}

func (t Token) isOperator() bool {
	return t == OP_EQI || t == OP_EQ || t == OP_NEQI || t == OP_NEQ || t == OP_RX || t == OP_RXN ||
		t == OP_GT || t == OP_GE || t == OP_LT || t == OP_LE
}

func (t Token) isLink() bool {
	return t == LNK_AND || t == LNK_OR
}

const (
	EOF = iota
	WS
	OP_EQI  // == equal ignore space
	OP_EQ   // === equal
	OP_NEQI // != not equal ignore space
	OP_NEQ  // !== not equal
	OP_RX   // ~= match regular expression
	OP_RXN  // ~! not match regular expression
	OP_GT   // >
	OP_GE   // >=
	OP_LT   // <
	OP_LE   // <=
	LNK_AND // &
	LNK_OR  // |
	LNK_NOT // NOT
	LITERAL
	INTEGER
	FLOAT
	BOOLEAN
	TIME
	ILLEGAL
)

const (
	eof     = rune(0)
	cEscape = '\\'
	cQuote  = '"'
	cSquote = '\''
	cEqual  = '='
	cExcl   = '!'
	cTilde  = '~'
	cGt     = '>'
	cLt     = '<'
	cAnd    = '&'
	cOr     = '|'
	sNot    = "not"
)

type Scanner struct {
	r *bufio.Reader
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

func (s Scanner) Scan() (Token, string) {
	c := s.read()
	if c == eof {
		return EOF, ""
	}

	switch c {
	case cEqual:
		c2 := s.read()
		if c2 == cEqual {
			c3 := s.read()
			if c3 == cEqual {
				return OP_EQ, "==="
			}
			s.unread()
			return OP_EQI, "=="
		}
		s.unread()
		return ILLEGAL, string(c)
	case cExcl:
		c2 := s.read()
		if c2 == cEqual {
			c3 := s.read()
			if c3 == cEqual {
				return OP_NEQ, "!=="
			}
			return OP_NEQI, "!="
		}
		s.unread()
		return ILLEGAL, string(c)
	case cTilde:
		c2 := s.read()
		if c2 == cEqual {
			return OP_RX, "~="
		} else if c2 == cExcl {
			return OP_RXN, "~!"
		}
		s.unread()
		return ILLEGAL, string(c)
	case cGt:
		c2 := s.read()
		if c2 == cEqual {
			return OP_GE, ">="
		}
		s.unread()
		return OP_GT, string(c)
	case cLt:
		c2 := s.read()
		if c2 == cEqual {
			return OP_LE, "<="
		}
		s.unread()
		return OP_LT, string(c)
	case cAnd:
		return LNK_AND, string(c)
	case cOr:
		return LNK_OR, string(c)
	}

	if unicode.IsSpace(c) {
		s.unread()
		return s.scanWhitespace()
	}

	if c == cQuote || c == cSquote {
		return s.scanQuotedLiteral(c)
	}

	s.unread()
	t, l := s.scanLiteral()
	if strings.ToLower(l) == sNot {
		return LNK_NOT, l
	}
	return t, l
}

func (s *Scanner) read() rune {
	if c, _, err := s.r.ReadRune(); err != nil {
		return eof
	} else {
		return c
	}
}

func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
}

func (s *Scanner) scanWhitespace() (t Token, l string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if c := s.read(); c == eof {
			break
		} else if !unicode.IsSpace(c) {
			s.unread()
			break
		} else {
			buf.WriteRune(c)
		}
	}
	return WS, buf.String()
}

func (s Scanner) scanLiteral() (Token, string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if c := s.read(); c == eof {
			break
		} else if unicode.IsSpace(c) {
			s.unread()
			break
		} else {
			buf.WriteRune(c)
		}
	}

	str := buf.String()
	return typedLiteral(str)
}

func (s Scanner) scanQuotedLiteral(q rune) (Token, string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if c := s.read(); c == eof {
			return ILLEGAL, buf.String()
		} else if c == q {
			break
		} else if c == cEscape {
			c2 := s.read()
			if isEscapeChar(c2) {
				buf.WriteRune(c2)
			} else {
				return ILLEGAL, buf.String()
			}
		} else {
			buf.WriteRune(c)
		}
	}
	str := buf.String()
	if q == cSquote {
		return TIME, str
	}
	return LITERAL, str
}

func (s Scanner) isEmpty() bool {
	return s.r.Size() == 0
}

func isEscapeChar(c rune) bool {
	return c == cEscape || c == cQuote || c == cSquote
}

func typedLiteral(str string) (Token, string) {
	num := true
	for i, c := range str {
		if i == 0 {
			if !unicode.IsDigit(c) && c != '.' && c != '-' {
				num = false
			}
		} else {
			if !unicode.IsDigit(c) && c != '.' {
				num = false
			}
		}
	}
	if num {
		c := strings.Count(str, ".")
		if c == 1 {
			return FLOAT, str
		} else if c == 0 {
			return INTEGER, str
		}
	}
	if _, err := strconv.ParseBool(str); err == nil {
		return BOOLEAN, str
	}
	return LITERAL, str
}

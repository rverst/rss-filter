package goql

import (
	"fmt"
	"io"
)

type Parser struct {
	s   *Scanner
	buf struct {
		token   Token
		literal string
		n       int
	}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		s: NewScanner(r),
	}
}

func (p *Parser) Parse() (Things, error) {

	ts := NewThings()
	for {
		thing := new(Thing)
		step := 0
		for {
			t, l := p.scanIgnoreWhitespace()
			if t == ILLEGAL {
				return nil, fmt.Errorf("illegal token: %s", l)
			} else if t == EOF {
				return ts, nil
			}
			switch step {
			case 0:
				if t.isLink() {
					thing.Link = t
					continue
				}
				if t == LNK_NOT {
					thing.Negate = true
					continue
				}
				if !t.isLiteral() {
					return nil, fmt.Errorf("key literal expected, got: (%s|%s)", t, l)
				}
				if l == "" {
					return nil, fmt.Errorf("empty literal now allowed as key")
				}
				thing.Key = l
			case 1:
				if !t.isOperator() {
					return nil, fmt.Errorf("operator expected, got: (%s|%s)", t, l)
				}
				thing.Operator = t
			case 2:
				if !t.isLiteral() {
					return nil, fmt.Errorf("expression literal expected, got: (%s|%s)", t, l)
				}
				thing.ExprType = t
				thing.Expression = l
			}
			step++
			if step > 2 {
				break
			}
		}
		ts.Add(thing)
		fmt.Printf("THING: %v\n", thing)
	}
}

func (p *Parser) scan() (t Token, l string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.token, p.buf.literal
	}

	t, l = p.s.Scan()
	p.buf.token, p.buf.literal = t, l
	return
}

func (p *Parser) unscan() {
	p.buf.n = 1
}

func (p *Parser) scanIgnoreWhitespace() (t Token, l string) {
	t, l = p.scan()
	if t == WS {
		t, l = p.scan()
	}
	return
}

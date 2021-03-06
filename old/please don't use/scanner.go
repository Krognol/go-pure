package pure

import (
	"bytes"
	"regexp"
	"strings"
)

const eof = byte(0)

type pureError struct {
	error
}

type scanner struct {
	buf   *bytes.Buffer
	index int

	line, col int
}

func newScanner(b []byte) *scanner {
	return &scanner{
		buf:   bytes.NewBuffer(b),
		index: -1,
		line:  0,
		col:   0,
	}
}

func (s *scanner) scan() (b byte) {
	if s.index >= len(s.buf.Bytes()) {
		s.buf.UnreadByte()
		return eof
	}
	b, _ = s.buf.ReadByte()

	if b == '\n' {
		s.line++
		s.col = 0
	}
	s.col++
	return
}

func (s *scanner) Peek() byte {
	b, _ := s.buf.ReadByte()
	s.buf.UnreadByte()
	return b
}

func (s *scanner) unread() {
	s.buf.UnreadByte()
	s.col--
}

func IsWhitespace(b byte) bool {
	return b == '\n' || b == '\r' || b == '\t' || b == ' '
}

func IsNumber(b byte) bool {
	return b >= '0' && b <= '9'
}

func IsAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func IsAlphaNum(b byte) bool {
	return IsNumber(b) || IsAlpha(b)
}

func SpecialCharacter(b byte) bool {
	return regexp.MustCompile("[<|>,;.:-_'*¨^~!§½\"@#£¤$%€&/{(\\[\\])}=+?´`]?").MatchString(string(b))
}

func (s *scanner) ScanIdentifier() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteByte(s.scan())
	tok = IDENTIFIER
	for {
		c := s.scan()

		if c == eof {
			return tok, buf.String()
		}

		if !IsAlphaNum(c) {
			if c == '.' || (IsWhitespace(c) && IsWhitespace(s.Peek())) {
				s.unread()
				return GROUP, buf.String()
			}

			if strings.ToLower(buf.String()) == "true" || strings.ToLower(buf.String()) == "false" {
				return BOOL, buf.String()
			}
			s.unread()
			return IDENTIFIER, buf.String()
		}

		buf.WriteByte(c)
	}
}

func (s *scanner) ScanNumber() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteByte(s.scan())
	tok = INT
	for {
		c := s.scan()

		if c == eof {
			return tok, buf.String()
		}

		if !IsNumber(c) {
			if c == '.' {
				tok = DOUBLE
				buf.WriteByte(c)
				continue
			}
			if IsAlpha(c) || SpecialCharacter(c) && (c != '\r' && c != '\n') {
				tok = QUANTITY
				buf.WriteByte(c)
				continue
			}
			s.unread()
			lit = buf.String()
			return
		}

		buf.WriteByte(c)
	}
}

func (s *scanner) ScanString() (tok Token, lit string) {
	var buf bytes.Buffer

	for c := s.scan(); c != '"'; c = s.scan() {
		if c == eof {
			return tok, buf.String()
		}

		if c == '\\' {
			if p := s.Peek(); p == '\n' || p == '\r' {
				for {
					c = s.scan()

					if IsWhitespace(c) {
						continue
					}
					s.unread()
					break
				}
			}
			buf.WriteByte(s.scan())
			continue
		}

		buf.WriteByte(c)
	}
	s.scan()
	return STRING, buf.String()
}

func (s *scanner) ScanPath() (tok Token, lit string) {
	var buf bytes.Buffer
	c := s.scan()
	buf.WriteByte(c) // consume the '.' or '/'

	for {
		c = s.scan()
		if c == eof {
			return tok, buf.String()
		}

		if !IsAlphaNum(c) {
			if c == '/' || c == '\\' || c == '.' || c == '-' || c == '_' || c == ' ' || c == ':' {
				buf.WriteByte(c)
				continue
			}
			s.unread()
			return PATH, buf.String()
		}
		buf.WriteByte(c)
	}
}

func (s *scanner) ScanEnv() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteByte(s.scan()) // consume the '$'
	for {
		c := s.scan()

		if c == eof {
			return tok, buf.String()
		}

		if !IsAlpha(c) {
			if c == '{' {
				buf.WriteByte(c)
				continue
			}
			if c == '}' {
				buf.WriteByte(c)
				return ENV, buf.String()
			}
			s.unread()
			return ENV, buf.String()
		}

		buf.WriteByte(c)
	}
}

func (s *scanner) ScanInclude() (tok Token, lit string) {
	var buf bytes.Buffer

	for {
		c := s.scan()

		if c == eof {
			return tok, buf.String()
		}

		if !IsAlphaNum(c) {
			if buf.String() == "include" {
				_, lit := s.ScanPath()
				buf.Reset()
				buf.WriteString(lit)
			}
			return INCLUDE, buf.String()
		}

		buf.WriteByte(c)
	}
}

func (s *scanner) ScanUnquotedString() (tok Token, lit string) {
	var buf bytes.Buffer
	if IsWhitespace(s.Peek()) {
		s.scan()
	}

	tok = STRING
	for c := s.scan(); c != '\r' && c != eof; c = s.scan() {
		if c == eof {
			return tok, buf.String()
		}

		if c == '\\' {
			if p := s.Peek(); p == '\n' || p == '\r' {
				for {
					c = s.scan()

					if IsWhitespace(c) {
						continue
					}
					s.unread()
					break
				}
			}
			buf.WriteByte(s.scan())
			continue
		}

		buf.WriteByte(c)
	}
	s.scan()
	return STRING, buf.String()
}

func (s *scanner) ConsumeComment() {
	for {
		c := s.scan()

		if c == '\r' || c == '\n' {
			break
		}
	}
}

func (s *scanner) Scan() (tok Token, lit string) {
	var buf bytes.Buffer
	c := s.scan()
	buf.WriteByte(c)

	if IsWhitespace(c) {
		return WHITESPACE, buf.String()
	}

	if IsAlpha(c) {
		s.unread()
		return s.ScanIdentifier()
	}

	if IsNumber(c) {
		s.unread()
		return s.ScanNumber()
	}

	switch c {
	case eof:
		return EOF, buf.String()
	case '"':
		return s.ScanString()
	case '.':
		if c = s.Peek(); c == '/' {
			s.unread()
			return s.ScanPath()
		}
		s.unread()
		return DOT, "."
	case '$':
		s.unread()
		return s.ScanEnv()
	case '%':
		return s.ScanInclude()
	case '[':
		return ARRAY, "["
	case ']':
		return RBRACK, "]"
	case '=':
		if s.Peek() == '>' {
			s.scan()
			return REF, "=>"
		}
		return EQUALS, "="
	case ':':
		return COLON, ":"
	case '/':
		return s.ScanPath()
	case '#':
		s.ConsumeComment()
		return s.Scan()
	}
	return Illegal, buf.String()
}

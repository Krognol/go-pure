package pure

import (
	"bytes"
)

const eof = byte(0)

type pureError struct {
    error
}

type scanner struct {
    buf *bytes.Buffer
    index int
    
    line, col int
}

func newScanner(b []byte) *scanner {
    return &scanner {
        buf: bytes.NewBuffer(b),
        index: -1,
        line: 0,
        col: 0,
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
    return
}

func (s *scanner) Peek() byte {
    b, _ := s.buf.ReadByte()
    s.buf.UnreadByte()
    return b
}

func (s *scanner) unread() {
    s.buf.UnreadByte()
    /*if s.col == 0 {
        str, _ := s.buf.ReadString('\n')
        s.col = len(str) - 1
        s.line--
    }*/
    s.col--
}

func IsWhitespace(b byte) bool {
    return b == '\n' || b == '\r' || b == '\t' || b == ' '
}

func IsNumber(b byte) bool {
    return b >= '0' && b <= '9'
}

func IsAlpha(b byte) bool {
    return b >= 'a' && b <= 'z'
}

func IsAlphaNum(b byte) bool {
    return IsNumber(b) || IsAlpha(b)
}

func (s *scanner) ScanIdentifier() (tok Token, lit string) {
    var buf bytes.Buffer
    buf.WriteByte(s.scan())

    for {
        c := s.scan()

        if c == eof {
            return EOF, "EOF"
        }

        if !IsAlpha(c) {
            if c == '.' || (IsWhitespace(c) && IsWhitespace(s.Peek())) {
                s.unread()
                return GROUP, buf.String()
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
            return EOF, "EOF"
        }

        if !IsNumber(c) {
            if c == '.' {
                tok = DOUBLE
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

    for c := s.scan(); c != '"'; c = s.scan(){
        if c == eof {
            return EOF, buf.String()
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
            return EOF, "EOF"
        }

        if !IsAlphaNum(c) {
            if c == '/' || c == '\\' || c == '.' {
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
    
    for {
        c := s.scan()

        if c == eof {
            return EOF, "EOF"
        }

        if !IsAlphaNum(c) {
            s.unread()
            return PATH, buf.String()
        }

        buf.WriteByte(c)
    }
}

func (s *scanner) ScanInclude() (tok Token, lit string) {
    var buf bytes.Buffer

    for {
        c := s.scan()

        if c == eof {
            return EOF, "EOF"
        }

        if !IsAlphaNum(c) {
            s.unread()
            return INCLUDE, buf.String()
        }

        buf.WriteByte(c)
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
        return EOF, "EOF"
        case '"':
        return s.ScanString()
        case '.':
        if c = s.Peek(); c == '/' {
            s.unread()
            s.unread()
            return s.ScanPath()
        } 
        s.unread()
        return DOT, "."
        case '$':
        return s.ScanEnv()
        case '%':
        return s.ScanInclude()
        case '[':
        return ARRAY, "["
        case '=':
        return EQUALS, "="
        case ':':
        return COLON, ":"
        case '/':
        return s.ScanPath()
    }
    return Illegal, buf.String()
}
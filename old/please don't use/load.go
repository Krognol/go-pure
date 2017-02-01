package pure

import "io/ioutil"

type pureGroup struct {
	properties map[string]pureProperty
}

type pureProperty struct {
	name  string
	value interface{}
	typ   Token
}

type reader struct {
	scanner *scanner
	tok     Token
	lit     string
}

type PureFile struct {
	value map[string]*pureProperty
}

func Load(file string) (*PureFile, error) {
	b, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, err
	}

	r := &reader{scanner: newScanner(b)}
	p := &PureFile{value: make(map[string]*pureProperty)}
	err = r.load(p)

	if err != nil {
		return nil, err
	}

	return p, err
}

func (r *reader) skipWhitespace() (tok Token, lit string) {
	for {
		tok, lit = r.scanner.Scan()

		if tok != WHITESPACE {
			return
		}
	}
}

func (r *reader) peek(n int) []byte {
	buf := r.scanner.buf

	return buf.Next(n)

}

func (r *reader) parseArray() *pureProperty {
	p := &pureProperty{}
	p.value = make([]interface{}, 0)

	for {
		r.tok, r.lit = r.skipWhitespace()

		if r.tok == EOF || r.tok == RBRACK {
			return p
		}

		if r.lit == "[" {
			for {
				r.tok, r.lit = r.skipWhitespace()

				if r.tok == GROUP || r.tok == IDENTIFIER {
					return r.keyValuePair()
				}

				break
			}

		}
	}
}

func (r *reader) keyValuePair() *pureProperty {
	return nil
}

func (r *reader) load(p *PureFile) error {
	for {
		r.tok, r.lit = r.skipWhitespace()

		if r.tok == EOF {
			return nil
		}

		switch r.tok {
		case IDENTIFIER:
			if r.tok, _ = r.skipWhitespace(); r.tok == EQUALS {
				if b := r.peek(2); b[0] == '[' || b[1] == '[' {
					id := r.lit
					p.value[id] = r.parseArray()
					continue
				}

			}
		}
	}
}

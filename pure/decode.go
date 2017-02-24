package pure

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

type Parser struct {
	buf                *bytes.Buffer
	start, end, actual int
	line, col          int
	src                []byte
}

func newParser(src []byte) *Parser {
	return &Parser{
		buf:  bytes.NewBuffer(src),
		line: 1,
		src:  src,
	}
}

func (p *Parser) reportErr(msg string) {
	var pointer string
	point := fmt.Sprintf("Error [%d : %d] : %s", p.line, p.col, string(p.src[p.start:p.end]))

	for i := 0; i < len(point)-1; i++ {
		pointer += "-"
	}
	pointer += "^"

	fmt.Printf("%s\n%s\n%s\n", point, pointer, msg)
}

// Shamelessly stolen from the Golang JSON decode source. Forgive
func indirect(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	for {
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && v.CanSet() {
			break
		}

		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if v.Type().NumMethod() > 0 {
			// TODO
		}

		v = v.Elem()
	}
	return v
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isAlNum(b byte) bool {
	// Should allow special characters too for identifiers
	return isAlpha(b) || isDigit(b) || b == '_'
}

func isWhiteSpace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\t'
}

func (p *Parser) consumeComment() {
	for {
		if p.getNext() == 10 {
			break
		}
	}
}

func getField(ident string, v reflect.Value) (reflect.Value, bool) {
	var isUnquoted bool
	lenIdent := len(ident)
	var iv reflect.Value
	var tag string
	if v.Kind() == reflect.Ptr {
		iv = indirect(v.Elem())
		for i := 0; i < iv.NumField(); i++ {
			tag = iv.Type().Field(i).Tag.Get("pure")

			if len(tag) >= lenIdent && tag[lenIdent:] == ",unquoted" {
				isUnquoted = true

				tag = tag[:lenIdent]

			}

			if tag == "" || tag == "-" || tag != ident {
				isUnquoted = false
				continue
			}

			return iv.Field(i), isUnquoted
		}
	}

	if v.Kind() == reflect.Struct {
		iv = indirect(v)
		tv := reflect.TypeOf(v.Interface())

		for i := 0; i < iv.NumField(); i++ {
			tag = tv.Field(i).Tag.Get("pure")

			if len(tag) >= lenIdent && tag[lenIdent:] == ",unquoted" {
				isUnquoted = true
				tag = tag[:lenIdent]
			}

			if tag == "" || tag == "-" || tag != ident {
				isUnquoted = false
				continue
			}

			return iv.Field(i), isUnquoted
		}
	}
	return indirect(v), isUnquoted
}

func (p *Parser) peek() byte {
	b, _ := p.buf.ReadByte()
	p.buf.UnreadByte()
	return b
}

// Peek the first n bytes in the buffer
func (p *Parser) peekn(n int) []byte {

	// Keep a backup of the current buffer
	backup := p.buf
	bs := [64]byte{}

	// Check the first n bytes
	for i := 0; i < n; i++ {
		b, _ := p.buf.ReadByte()
		bs[i] = b
	}

	// Rewind
	p.buf = backup
	return bs[:n]
}

func (p *Parser) getValue() []byte {
	var buf = bytes.NewBuffer(nil)
	var b byte
	// Skip the leading whitespaces
	for {
		if isWhiteSpace(p.getNext()) {
			continue
		}
		p.buf.UnreadByte()
		p.actual--
		break
	}

	// Grab any byte until we hit a new line
	for {
		b = p.getNext()

		if b == '\\' {
			peek := p.peek()
			if peek == ' ' || peek == '\r' || peek == '\n' {
				p.getNext()
				p.getNext()
				for {
					if !isWhiteSpace(p.getNext()) {
						p.buf.UnreadByte()
						break
					}
				}
				continue
			}
		}

		if b == 0 || b == 10 {
			break
		}

		buf.WriteByte(b)
	}
	return buf.Bytes()
}

func (p *Parser) verifyValue(value string) string {
	var buf = bytes.NewBuffer(nil)

	for i := 0; i < len(value); i++ {

		// Character escape
		if value[i] == '\\' {
			i++
			if i > len(value) {
				i--
				break
			}
		}

		buf.WriteByte(value[i])
	}
	return buf.String()
}

func (p *Parser) fieldSetValue(field reflect.Value, val string, unq bool) error {

	val = strings.Replace(val, "\r", "", -1)
	switch field.Kind() {
	case reflect.Int:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.String:
		if unq || val[0] != '"' {
			field.SetString(p.verifyValue(val))
			break
		}
		field.SetString(p.verifyValue(val[1 : len(val)-1]))
	case reflect.Bool:
		b, err := strconv.ParseBool(strings.ToLower(val))
		if err != nil {
			return err
		}
		field.SetBool(b)
	}
	return nil
}

func (p *Parser) parseMap(v reflect.Value) (reflect.Value, error) {
	// We've already consumed the bracket
	// so just start getting the keys and values
	// straight away
	var buf = bytes.NewBuffer(nil)

	iv := indirect(v)
	for b := p.getNext(); b != ']'; b = p.getNext() {
		if isWhiteSpace(b) || b == '\r' {
			continue
		}

		if b == 0 {
			p.end = p.actual - 1
			return iv, fmt.Errorf("Invalid map property, missing ']'")
		}

		if !isAlNum(b) {

			// TODO :: Add map value referencing
			// TODO :: Add map groups
			if b == '=' {

				value := bytes.Replace(p.getValue(), []byte("\r"), nil, -1)
				var mval reflect.Value
				switch iv.Type().Elem().Kind() {
				case reflect.Int:
					i, err := strconv.Atoi(string(value))
					if err != nil {
						p.end = p.actual - 1
						return iv, err
					}
					mval = reflect.ValueOf(i)
				case reflect.Float64:
					f, err := strconv.ParseFloat(string(value), 64)
					if err != nil {
						p.end = p.actual - 1
						return iv, err
					}
					mval = reflect.ValueOf(f)
				case reflect.Bool:
					bol, err := strconv.ParseBool(string(value))
					if err != nil {
						p.end = p.actual - 1
						return iv, err
					}
					mval = reflect.ValueOf(bol)
				case reflect.String:
					mval = reflect.ValueOf(string(value))
				default:
					p.end = p.actual - 1
					return iv, fmt.Errorf("Invalid type %s", iv.Kind().String())
				}
				iv.SetMapIndex(reflect.ValueOf(buf.String()), mval)
				buf.Reset()
				continue
			}
		}
		buf.WriteByte(b)
	}
	return iv, nil
}

func (p *Parser) getNext() byte {
	b, _ := p.buf.ReadByte()
	p.col++
	p.actual++

	if b == 10 {
		p.line++
		p.col = 0
	}

	if b == 0 {
		p.col--
		p.actual--
	}
	return b
}

func (p *Parser) parseArray(v reflect.Value) (reflect.Value, error) {
	// Consume the '['
	p.getNext()
	var val []byte
	value := indirect(v)

	if value.Kind() == reflect.Map {
		return p.parseMap(v)
	}

	for b := p.getNext(); b != ']'; b = p.getNext() {
		val = bytes.Replace(p.getValue(), []byte("\r"), nil, -1)
		var app reflect.Value
		switch value.Type().Elem().Kind() {
		case reflect.Int:
			i, err := strconv.Atoi(string(val))
			if err != nil {
				return value, err
			}
			app = reflect.ValueOf(i)
		case reflect.Float64:
			f, err := strconv.ParseFloat(string(val), 64)
			if err != nil {
				return value, err
			}
			app = reflect.ValueOf(f)
		case reflect.Bool:
			bol, err := strconv.ParseBool(string(val))
			if err != nil {
				return value, err
			}
			app = reflect.ValueOf(bol)
		case reflect.String:
			app = reflect.ValueOf(string(val))
		default:
			return value, fmt.Errorf("Invalid type %s", value.Kind().String())
		}
		value = reflect.Append(value, app)
	}
	return value, nil
}

func (p *Parser) parseIdent(v reflect.Value) error {
	var b byte
	var buf = bytes.NewBuffer(nil)
	backup := v
	p.start = p.actual
	// While the current byte is a letter or number
	// assume it's part of the identifier
	for {
		b = p.getNext()

		if b == 0 {
			p.end = p.actual - 1
			p.reportErr("Identifier missing value")
			return nil
		}

		if isWhiteSpace(b) {
			continue
		}

		if !isAlNum(b) {
			p.buf.UnreadByte()
			p.actual--
			break
		}

		buf.WriteByte(b)
	}

	// Skip trailing whitespaces until we hit a '='
	for {
		b = p.getNext()

		if b == 0 {
			p.end = p.actual - 1
			p.reportErr("Identifier missing value")
			break
		}

		if b == 10 {
			for {
				if p.peek() == '\t' || p.peek() == ' ' {
					field, _ := getField(buf.String(), v)
					p.parseIdent(field)
					continue
				}
				break
			}
			break
		}

		if isWhiteSpace(b) {
			continue
		}

		// We're assigning a group variable
		if b == '.' {
			group, _ := getField(buf.String(), v)
			buf.Reset()
			for {
				b = p.getNext()
				if b == 0 {
					p.end = p.actual - 1
					p.reportErr("Missing group variable identifier")
					break
				}

				if !isAlNum(b) {
					p.buf.UnreadByte()
					p.actual--
					break
				}

				buf.WriteByte(b)
			}

			//ident = field
			v = group
		}

		if b == '=' {
			// Get the field from the struct that has a tag that matches
			// ident
			field, unquoted := getField(buf.String(), v)
			peek := p.peek()
			if isWhiteSpace(peek) {
				p.getNext()
				peek = p.peek()
			}

			if peek == '[' {
				values, err := p.parseArray(field)
				if err != nil {
					p.end = p.actual - 1
					p.reportErr(err.Error())
					return err
				}

				field.Set(values)
				return nil
			}

			// Check for reference values
			if peek == '>' {
				// Consume the '>'
				p.getNext()
				return p.parseReference(field, backup)

			}

			// Get the value of the field and set it
			// Throws an error if the value is invalid
			val := p.getValue()
			err := p.fieldSetValue(field, string(val), unquoted)
			if err != nil {
				p.end = p.actual - 1
				p.reportErr("Couldn't set field value " + string(val))
				return err
			}
			break
		}

	}
	return nil
}

func (p *Parser) parseReference(field reflect.Value, v reflect.Value) error {
	getFrom := p.getValue()
	getFrom = bytes.Replace(getFrom, []byte("\r"), nil, -1)

	if index := bytes.IndexByte(getFrom, '.'); index != -1 {
		group, _ := getField(string(getFrom[:index]), v)
		getFrom = getFrom[index+1:]
		v = group
	}

	fromField, _ := getField(string(getFrom), v)
	var value string

	switch fromField.Kind() {
	case reflect.Int:
		value = strconv.Itoa(int(fromField.Int()))
	case reflect.Float64:
		value = strconv.FormatFloat(fromField.Float(), 'f', 12, 64)
	case reflect.Bool:
		value = strconv.FormatBool(fromField.Bool())
	default:
		value = fromField.String()
	}

	err := p.fieldSetValue(field, value, false)
	if err != nil {
		p.end = p.actual - 1
		p.reportErr("Couldn't set field value " + value)
		return err
	}
	return nil
}

func (p *Parser) parseInclude(v interface{}) error {
	var buf = bytes.NewBuffer(nil)

	for {
		b := p.getNext()

		if b == 0 || b == 10 {
			p.end = p.actual - 1
			p.reportErr("No include specified")
			return nil
		}

		buf.WriteByte(b)

		if buf.String() == "include" {
			buf.Reset()
			for {
				b = p.getNext()

				if b == 0 {
					break
				}

				if isWhiteSpace(b) {
					continue
				}

				if !isAlNum(b) {
					if b == '.' || b == '/' || b == '\\' {
						buf.WriteByte(b)
						continue
					}
					break
				}
				buf.WriteByte(b)
			}
			break
		}
	}
	f, err := ioutil.ReadFile(buf.String())
	if err != nil {
		p.end = p.actual
		p.reportErr("Couldn't open file '" + buf.String() + "'")
		return err
	}
	return Unmarshal(f, v)
}

func (p *Parser) unmarshal(v interface{}) error {
	// While the current byte is not 0, we advance through the buffer
	for b := p.getNext(); b != 0; b = p.getNext() {

		if b == 0 {
			break
		}

		if b == 10 {
			continue
		}

		// Ignore all initial whitespace
		if isWhiteSpace(b) {
			continue
		}

		// Consume comments
		if b == '#' {
			p.consumeComment()
			continue
		}

		// If we encounter a letter assume that it's an identifier with a value
		if isAlpha(b) {
			p.buf.UnreadByte()
			p.actual--
			err := p.parseIdent(reflect.ValueOf(v))
			if err != nil {
				return err
			}
		}

		if b == '%' {
			p.start = p.actual - 1
			err := p.parseInclude(v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Unmarshal decodes a Pure source into a golang struct
func Unmarshal(src []byte, v interface{}) error {
	parser := newParser(src)

	// Make sure the supplied type is a pointer
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return hasToBePtrTypeError(v)
	}

	return parser.unmarshal(v)
}

func hasToBePtrTypeError(v interface{}) error {
	return fmt.Errorf("%s has to be of pointer type\n", reflect.ValueOf(v).Type().Name())
}

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
	buf        *bytes.Buffer
	start, end int
	line, col  int
	src        []byte
}

func NewParser(src []byte) *Parser {
	return &Parser{
		buf:   bytes.NewBuffer(src),
		line:  1,
		col:   0,
		start: 0,
		end:   0,
		src:   src,
	}
}

func (p *Parser) reportErr(msg string) {
	fmt.Printf("Error [%d : %d] : %s\n%s\n", p.line, p.col, string(p.src[p.start:p.end]), msg)
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
	return isAlpha(b) || isDigit(b) || b == '_'
}

func isWhiteSpace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\t'
}

func (p *Parser) consumeComment() {
	for {
		b, _ := p.buf.ReadByte()
		if b == 10 {
			p.line++
			p.col = 0
			break
		}
		p.col++
		p.end++
	}
}

func getField(ident string, v reflect.Value) (reflect.Value, bool) {
	var isUnquoted bool
	if v.Kind() == reflect.Ptr {
		iv := indirect(v.Elem())
		for i := 0; i < iv.NumField(); i++ {
			tag := iv.Type().Field(i).Tag.Get("pure")
			if split := strings.Split(tag, ","); len(split) > 1 {
				if split[1] == "unquoted" {
					isUnquoted = true
				}
				tag = split[0]
			}

			if tag == "" || tag == "-" || tag != ident {
				isUnquoted = false
				continue
			}

			return iv.Field(i), isUnquoted
		}
	}

	if v.Kind() == reflect.Struct {
		iv := indirect(v)
		tv := reflect.TypeOf(v.Interface())

		for i := 0; i < iv.NumField(); i++ {
			tag := tv.Field(i).Tag.Get("pure")

			if split := strings.Split(tag, ","); len(split) > 1 {
				if split[1] == "unquoted" {
					isUnquoted = true
				}
				tag = split[0]
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

func (p *Parser) peekn(n int) []byte {
	backup := p.buf
	var bs []byte
	for i := 0; i < n; i++ {
		b, _ := p.buf.ReadByte()
		bs = append(bs, b)
	}
	p.buf = backup
	return bs
}

func (p *Parser) getValue() string {
	var s string

	// Skip the leading whitespaces
	for {
		if b, _ := p.buf.ReadByte(); isWhiteSpace(b) {
			p.col++
			p.end++
			continue
		}
		p.buf.UnreadByte()
		break
	}

	// Grab any byte until we hit a new line
	for {
		b, _ := p.buf.ReadByte()

		if b == 0 {
			return s
		}

		if b == 10 {
			p.line++
			p.col = 0
			break
		}

		s += string(b)
	}
	return s
}

func (p *Parser) verifyValue(value string) string {
	var s string

	for i := 0; i < len(value); i++ {

		// Character escape
		if value[i] == '\\' {
			i++
		}

		if value[i] == ' ' && i+1 < len(value) {
			if value[i+1] == '\r' {
				for {
					i += 2
					if !isWhiteSpace(value[i]) {
						break
					}
				}
			}
		}
		s += string(value[i])
	}
	return s
}

func (p *Parser) fieldSetValue(field reflect.Value, val string, unq bool) error {
	// For now we remove all carriage returns, but should make a string value verifier
	// in the future
	val = strings.Replace(val, "\r", "", -1)
	switch field.Kind() {
	case reflect.Int:
		i, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		field.SetInt(int64(i))
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

func (p *Parser) parseMap(v interface{}) (reflect.Value, error) {
	// We're already consumed the bracket
	// so just start getting the keys and values
	// straight away

	var (
		key, value string
	)
	iv := indirect(reflect.ValueOf(v))
	for b, _ := p.buf.ReadByte(); b != ']'; b, _ = p.buf.ReadByte() {
		if isWhiteSpace(b) || b == '\r' {
			continue
		}

		if b == 0 {
			return iv, fmt.Errorf("Invalid map, missing ']'")
		}

		if !isAlNum(b) {

			// For now maps won't support referencing
			// TODO :: Add map value referencing
			if b == '=' {
				value = strings.Replace(p.getValue(), "\r", "", -1)

				switch iv.Type().Elem().Kind() {
				case reflect.Int:
					i, err := strconv.Atoi(value)
					if err != nil {
						return iv, err
					}
					iv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(i))
				case reflect.Float64:
					f, err := strconv.ParseFloat(value, 64)
					if err != nil {
						return iv, err
					}
					iv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(f))
				case reflect.Bool:
					bol, err := strconv.ParseBool(value)
					if err != nil {
						return iv, err
					}
					iv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(bol))
				case reflect.String:
					iv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
				default:
					return iv, fmt.Errorf("Invalid type %s", iv.Kind().String())
				}
				key, value = "", ""
				continue
			}
		}
		key += string(b)
	}
	return iv, nil
}

func (p *Parser) parseArray(v interface{}) (reflect.Value, error) {
	// Consume the '['
	p.buf.ReadByte()

	value := indirect(reflect.ValueOf(v))
	if value.Kind() == reflect.Map {
		return p.parseMap(v)
	}

	for b, _ := p.buf.ReadByte(); b != ']'; b, _ = p.buf.ReadByte() {
		val := p.getValue()
		val = strings.Replace(val, "\r", "", -1)
		switch value.Type().Elem().Kind() {
		case reflect.Int:
			i, err := strconv.Atoi(val)
			if err != nil {
				return value, err
			}
			value = reflect.Append(value, reflect.ValueOf(i))
		case reflect.Float64:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return value, err
			}
			value = reflect.Append(value, reflect.ValueOf(f))
		case reflect.Bool:
			bol, err := strconv.ParseBool(val)
			if err != nil {
				return value, err
			}
			value = reflect.Append(value, reflect.ValueOf(bol))
		case reflect.String:
			value = reflect.Append(value, reflect.ValueOf(val))
		default:
			return value, fmt.Errorf("Invalid type %s", value.Kind().String())
		}
	}
	return value, nil
}

func (p *Parser) parseIdent(v interface{}) error {
	var ident string
	backup := v
	// While the current byte is a letter or number
	// assume it's part of the identifier
	for {
		b, err := p.buf.ReadByte()
		p.col++
		p.end++
		p.start = p.col

		if err != nil {
			p.reportErr("Buffer read error")
			return err
		}

		if b == 0 {
			return nil
		}

		if isWhiteSpace(b) {
			continue
		}

		if !isAlNum(b) {
			p.buf.UnreadByte()
			p.col--
			break
		}

		ident += string(b)
	}

	// Skip trailing whitespaces until we hit a '='
	for {
		b, _ := p.buf.ReadByte()
		p.col++
		p.end++
		if b == 0 {
			p.end = p.col - 1
			break
		}

		if isWhiteSpace(b) {
			continue
		}

		if b == 10 {
			err := p.parseIdent(v)
			if err != nil {
				return err
			}
		}

		// We're assigning a group variable
		if b == '.' {
			group, _ := getField(ident, reflect.ValueOf(v))
			var field string
			for {
				b, _ := p.buf.ReadByte()
				if b == 0 {
					break
				}

				if !isAlNum(b) {
					p.buf.UnreadByte()
					break
				}

				field += string(b)
			}

			ident = field
			v = group.Interface()
		}

		if b == '=' {
			// Get the field from the struct that has a tag that matches
			// ident
			field, unquoted := getField(ident, reflect.ValueOf(v))

			if isWhiteSpace(p.peek()) {
				p.buf.ReadByte()
			}

			if p.peek() == '[' {
				values, err := p.parseArray(field.Interface())
				if err != nil {
					return err
				}

				field.Set(values)
				return nil
			}

			// Check for reference values
			if p.peek() == '>' {
				// Consume the '>'
				p.buf.ReadByte()
				p.col++
				p.end++
				getFrom := p.getValue()
				getFrom = strings.Replace(getFrom, "\r", "", -1)

				if split := strings.Split(getFrom, "."); len(split) > 1 {
					group, _ := getField(split[0], reflect.ValueOf(backup))
					getFrom = split[1]
					v = group.Interface()
				}

				fromField, _ := getField(getFrom, reflect.ValueOf(v))
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
					p.reportErr("Couldn't set field value " + value)
					return err
				}
				break
			}

			// Get the value of the field and set it
			// Throws an error if the value is invalid
			val := p.getValue()
			err := p.fieldSetValue(field, val, unquoted)
			if err != nil {
				p.reportErr("Couldn't set field value " + val)
				return err
			}
			break
		}

		if p.peek() == 10 {
			p.line++
			for {
				peek := p.peekn(2)
				if peek[1] == '\t' || peek[1] == ' ' {
					field, _ := getField(ident, reflect.ValueOf(v))
					p.parseIdent(field.Interface())
					continue
				}
				break
			}
			break
		}
	}
	return nil
}

func (p *Parser) parseInclude(v interface{}) error {
	var doInclude string
	for {
		b, err := p.buf.ReadByte()
		p.col++
		p.end++
		p.start = p.col
		if err != nil {
			p.reportErr("Buffer read error")
			return err
		}

		if b == 0 || b == 10 {
			return nil
		}

		doInclude += string(b)

		if doInclude == "include" {
			doInclude = ""

			for {
				b, err = p.buf.ReadByte()

				if b == 0 {
					break
				}

				if isWhiteSpace(b) {
					continue
				}

				if !isAlNum(b) {
					if b == '.' || b == '/' || b == '\\' {
						doInclude += string(b)
						continue
					}
					break
				}
				doInclude += string(b)
			}

			break
		}
	}
	f, err := ioutil.ReadFile(doInclude)
	if err != nil {
		return err
	}
	return Unmarshal(f, v)
}

func (p *Parser) unmarshal(v interface{}) error {
	// While the current byte is not 0, we advance through the buffer
	for b, err := p.buf.ReadByte(); b != 0; b, err = p.buf.ReadByte() {
		p.col++
		p.end++
		if err != nil {
			p.reportErr("Buffer read error")
			return err
		}

		if b == 0 {
			break
		}

		if b == 10 {
			p.line++
			p.col = 0
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
			err = p.parseIdent(v)
			if err != nil {
				return err
			}
		}

		if b == '%' {
			err = p.parseInclude(v)
			if err != nil {
				p.reportErr("Couldn't include file")
				return err
			}
		}
	}
	return nil
}

// Unmarshal decodes a Pure source into a golang struct
func Unmarshal(src []byte, v interface{}) error {
	parser := NewParser(src)

	// Make sure the supplied type is a pointer
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return hasToBePtrTypeError(v)
	}

	return parser.unmarshal(v)
}

func hasToBePtrTypeError(v interface{}) error {
	return fmt.Errorf("%s has to be of pointer type\n", reflect.ValueOf(v).Type().Name())
}

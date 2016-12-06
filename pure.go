package pure

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
)

type state int

const tagName = "pure"

type unmarshaler struct {
	Scanner  *scanner
	errors   []*pureError
	tagID    string
	tagValue string
	tagTok   Token
	tagTyp   string
}

// Shamelessly stolen from the Golang JSON decode source. Forgive
func (u *unmarshaler) indirect(v reflect.Value) reflect.Value {
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

func (u *unmarshaler) newError(msg string) *pureError {
	s := fmt.Sprintf("Error unmarhsaling Pure property: %s\r\n[%d:%d]-%s", msg, u.Scanner.line, u.Scanner.col, string(u.Scanner.buf.Bytes()))
	err := &pureError{}
	err.error = fmt.Errorf(s)
	return err
}

func (u *unmarshaler) Scan() (tok Token, lit string) {
	return u.Scanner.Scan()
}

func (u *unmarshaler) ScanSkipWhitespace() (tok Token, lit string) {
	for tok, lit = u.Scanner.Scan(); tok == WHITESPACE; {
		tok, lit = u.Scanner.Scan()
	}
	return
}

func (u *unmarshaler) field(v reflect.Value) *pureError {
	var field reflect.Value

	switch v.Kind() {
	case reflect.Ptr:
		iv := u.indirect(v.Elem())
		for i := 0; i < iv.NumField(); i++ {
			tag := iv.Type().Field(i).Tag.Get(tagName)
			if tag != "" && tag != "-" && tag == u.tagID {
				field = iv.Field(i)
				break
			}
		}
	case reflect.Struct:
		iv := u.indirect(v)
		tv := reflect.TypeOf(v.Interface())

		for i := 0; i < iv.NumField(); i++ {
			tag := tv.Field(i).Tag.Get(tagName)
			if tag != "" && tag != "-" && tag == u.tagID {
				field = iv.Field(i)
				break
			}

		}
	}

	if !field.IsValid() {
		field = u.indirect(v)
	}

	switch {
	case field.Kind() == reflect.Int && u.tagTyp == "int":
		_i, err := strconv.Atoi(u.tagValue)
		if err != nil {
			return u.newError(fmt.Sprintf("bad number value '%s'", u.tagValue))
		}
		field.SetInt(int64(_i))
		return nil
	case field.Kind() == reflect.String && (u.tagTyp == "string" || u.tagTyp == "quantity" || u.tagTyp == "path"):
		field.SetString(u.tagValue)
		return nil
	case field.Kind() == reflect.Float64 && u.tagTyp == "double":
		f, err := strconv.ParseFloat(u.tagValue, 64)
		if err != nil {
			return u.newError(fmt.Sprintf("bad floating point value '%s'", u.tagValue))
		}
		field.SetFloat(f)
		return nil
	case field.Kind() == reflect.Bool && u.tagTyp == "bool":
		b, err := strconv.ParseBool(u.tagValue)
		if err != nil {
			return u.newError(fmt.Sprintf("bad bool value '%s'", u.tagValue))
		}
		field.SetBool(b)
		return nil
	case field.Kind() == reflect.Ptr && u.tagTyp == "group":
		return u.group(field.Interface())
	default:
		fi := field.Interface()
		switch fi.(type) {
		case *Quantity:
			if u.tagTyp != "quantity" {
				fmt.Println(u.newError(fmt.Sprintf("mismatched types 'Quantity' & '%s", u.tagTyp)))
				return nil
			}
			fi = NewQuantity(u.tagValue)
			field.Set(reflect.ValueOf(fi))
			return nil
		case Quantity:
			if u.tagTyp != "quantity" {
				fmt.Println(u.newError(fmt.Sprintf("mismatched types 'Quantity' & '%s", u.tagTyp)))
				return nil
			}
			fi = NewQuantity(u.tagValue)
			field.Set(u.indirect(reflect.ValueOf(fi)))
			return nil
		case *Env:
			if u.tagTyp != "env" {
				fmt.Println(u.newError(fmt.Sprintf("mismatched types 'Env' & '%s'", u.tagTyp)))
				return nil
			}
			fi = NewEnv(u.tagValue)
			field.Set(reflect.ValueOf(fi))
			return nil
		case Env:
			if u.tagTyp != "env" {
				fmt.Println(u.newError(fmt.Sprintf("mismatched types 'Env' & '%s'", u.tagTyp)))
				return nil
			}
			fi = NewEnv(u.tagValue)
			field.Set(u.indirect(reflect.ValueOf(fi)))
			return nil
		case *Path:
			if u.tagTyp != "path" {
				fmt.Println(u.newError(fmt.Sprintf("mismatched types 'Path' & '%s'", u.tagTyp)))
				return nil
			}
			fi = NewPath(u.tagValue)
			field.Set(reflect.ValueOf(fi))
			return nil
		case Path:
			if u.tagTyp != "path" {
				fmt.Println(u.newError(fmt.Sprintf("mismatched types 'Path' & '%s'", u.tagTyp)))
				return nil
			}
			fi = NewPath(u.tagValue)
			field.Set(u.indirect(reflect.ValueOf(fi)))
			return nil
		}

	}
	return nil
}

// Peek copies the unmarshalers buffer (to not advance the buffer we're reading form)
// and returns the next n bytes
func (u *unmarshaler) Peek(n int) []byte {
	return bytes.NewBuffer(u.Scanner.buf.Bytes()).Next(n)
}

// This is not pretty, but it works ¯\_(ツ)_/¯
func (u *unmarshaler) PeekLiteral() string {
	buf := bytes.NewBuffer(u.Scanner.buf.Bytes())
	for {
		b, _ := buf.ReadByte()

		if IsAlpha(b) {
			buf.WriteByte(b)
			for {
				b, _ := buf.ReadByte()
				if IsWhitespace(b) {
					break
				}
				buf.WriteByte(b)
			}
			break
		}
	}
	return buf.String()
}

func (u *unmarshaler) group(v interface{}) *pureError {
	iv := u.indirect(reflect.ValueOf(v))
	tv := reflect.TypeOf(v)
	for i := 0; i < iv.NumField(); i++ {
		tag := tv.Elem().Field(i).Tag.Get(tagName)

		if tag == u.tagID {
			f := iv.Field(i)
			for {
				tok, lit := u.Scan()
				if tok == EOF {
					return nil
				}

				if lit == "\r" {
					if b := u.Peek(2); b[0] == '\n' && (IsWhitespace(b[len(b)] - 1)) {
						continue
					}
					return nil
				}

				if lit == " " || lit == "\n" || lit == "\t" {
					continue
				}
				if tok == DOT || lit == "." {
					tok, lit = u.ScanSkipWhitespace()
				}

				if tok == GROUP {
					struc := u.GetStruct(u.tagID, v)
					field := u.GetField(lit, struc)
					u.tagID = u.PeekLiteral()
					err := u.group(field.Interface())
					if err != nil {
						fmt.Println(err.Error())
					}
				}

				u.tagID = lit

				tok, lit = u.ScanSkipWhitespace()

				if tok == EQUALS {
					u.tagTok, u.tagValue = u.ScanSkipWhitespace()
				}

				switch u.tagTok {
				case STRING, IDENTIFIER:
					u.tagTyp = "string"
				case INT:
					u.tagTyp = "int"
				case DOUBLE:
					u.tagTyp = "double"
				case BOOL:
					u.tagTyp = "bool"
				case QUANTITY:
					u.tagTyp = "quantity"
				case ENV:
					u.tagTyp = "env"
				case PATH:
					u.tagTyp = "path"
				}

				err := u.field(f)
				if err != nil {
					fmt.Println(err.Error())
				}
			}
		}
	}
	return nil
}

func (u *unmarshaler) GetStruct(name string, v interface{}) reflect.Value {
	iv := u.indirect(reflect.ValueOf(v))
	for i := 0; i < iv.NumField(); i++ {
		tag := reflect.TypeOf(v).Elem().Field(i).Tag.Get(tagName)
		if tag == name {
			return iv.Field(i)
		}
	}
	return reflect.Zero(nil)
}

func (u *unmarshaler) GetField(name string, v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		iv := u.indirect(v.Elem())

		for i := 0; i < iv.NumField(); i++ {
			tag := iv.Type().Field(i).Tag.Get(tagName)

			if tag == name {
				return iv.Field(i)
			}
		}
	}

	if v.Kind() == reflect.Struct {
		iv := u.indirect(v)
		tv := reflect.TypeOf(v.Interface())
		for i := 0; i < iv.NumField(); i++ {
			tag := tv.Field(i).Tag.Get(tagName)
			if tag == "" {
				tag = iv.Type().Field(i).Tag.Get(tagName)
			}
			if tag == name || tag == u.tagID {
				if iv.Kind() == reflect.Struct || iv.Kind() == reflect.Ptr {
					return u.GetField(u.tagID, reflect.ValueOf(iv.Field(i)))
				}
				return iv.Field(i)
			}
		}
	}

	return v
}

// Gotta pretty this up it's really ugly
// Makes me wanna vomit
func (u *unmarshaler) unmarshal(v interface{}) {
	pv := u.indirect(reflect.ValueOf(v))
	for {
		tok, lit := u.ScanSkipWhitespace()
		u.tagID = lit
		u.tagTok = tok

		if tok == EOF {
			return
		}

		switch tok {
		case IDENTIFIER:
			if tok, _ := u.ScanSkipWhitespace(); tok == EQUALS {
				u.tagTok, u.tagValue = u.ScanSkipWhitespace()
				switch u.tagTok {
				case INT:
					u.tagTyp = "int"
				case DOUBLE:
					u.tagTyp = "double"
				case STRING, IDENTIFIER:
					u.tagTyp = "string"
				case BOOL:
					u.tagTyp = "bool"
				case QUANTITY:
					u.tagTyp = "quantity"
				case ENV:
					u.tagTyp = "env"
				case PATH:
					u.tagTyp = "path"
				}
			} else if tok == REF {
				var field reflect.Value
				temp := lit
				tok, lit = u.ScanSkipWhitespace()
				if b := u.Peek(1); b[0] == '.' {
					group := lit
					tok, lit = u.ScanSkipWhitespace()
					tok, lit = u.ScanSkipWhitespace()
					u.tagID = lit
					struc := u.GetStruct(group, v)
					u.tagID = temp
					field = u.GetField(lit, struc)
				} else {
					// Assume it's a regular property and not a group property
					tok, lit = u.ScanSkipWhitespace()
					field = u.GetField(u.tagID, u.indirect(reflect.ValueOf(v)))
				}
				switch field.Kind() {
				case reflect.Int:
					u.tagTyp = "int"
					u.tagValue = strconv.Itoa(int(field.Int()))
				case reflect.Float64:
					u.tagTyp = "double"
					u.tagValue = strconv.FormatFloat(field.Float(), 'f', 16, 64)
				case reflect.String:
					u.tagTyp = "string"
					u.tagValue = field.String()
				case reflect.Bool:
					u.tagTyp = "bool"
					u.tagValue = strconv.FormatBool(field.Bool())
				default:
					fi := field.Interface()

					switch fi.(type) {
					case *Quantity:
						u.tagTyp = "quantity"
						u.tagValue = fi.(*Quantity).value
					case Quantity:
						u.tagTyp = "quantity"
						u.tagValue = fi.(Quantity).value
					case *Env:
						u.tagTyp = "env"
						u.tagValue = fi.(*Env).value
					case Env:
						u.tagTyp = "env"
						u.tagValue = fi.(Env).value
					case *Path:
						u.tagTyp = "path"
						u.tagValue = fi.(*Path).value
					case Path:
						u.tagTyp = "Path"
						u.tagValue = fi.(Path).value
					}
				}
			}

			err := u.field(pv)
			if err != nil {
				u.errors = append(u.errors, err)
			}

		case GROUP:
			err := u.group(v)
			if err != nil {
				u.errors = append(u.errors, err)
			}
		case INCLUDE:
			b, err := ioutil.ReadFile(lit)
			if err != nil {
				fmt.Println(u.newError(err.Error()).Error())
			}
			Unmarshal(b, v)
		}
	}
}

func Unmarshal(b []byte, v interface{}) *pureError {
	u := &unmarshaler{
		Scanner: newScanner(b),
	}
	u.unmarshal(v)

	// Should improve error reporting
	// Maybe as soon as they're discovered?
	if len(u.errors) > 0 {
		return u.errors[0]
	}
	return nil
}

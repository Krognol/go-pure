package pure

import (
	"fmt"
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
	s := fmt.Sprintf("Error unmarhsaling Pure property: %s\r\n[%d:%d]-%s", msg, u.Scanner.line, u.Scanner.col, u.Scanner.buf.String())
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
	if v.Kind() == reflect.Ptr {
		iv := u.indirect(v.Elem())
		for i := 0; i < iv.NumField(); i++ {
			tag := iv.Type().Field(i).Tag.Get(tagName)
			field := iv.Field(i)

			if tag != "" && tag != "-" && tag == u.tagID {
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
				}
			}
		}
	}

	iv := u.indirect(v)
	tv := reflect.TypeOf(v.Interface())
	println(iv.Kind().String())

	for i := 0; i < iv.NumField(); i++ {
		tag := tv.Field(i).Tag.Get(tagName)
		field := iv.Field(i)

		if tag != "" && tag != "-" && tag == u.tagID {
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
			}
		}
	}
	return nil
}

func (u *unmarshaler) group(v interface{}) *pureError {
	iv := u.indirect(reflect.ValueOf(v))
	tv := reflect.TypeOf(v)

	for i := 0; i < iv.NumField(); i++ {
		tag := tv.Elem().Field(i).Tag.Get(tagName)

		if tag == u.tagID {
			for {
				tok, lit := u.Scan()
				if tok == EOF {
					return nil
				}
				if lit == "\r" {
					return nil
				}

				if lit == " " || lit == "" || lit == "\n" || lit == "\t" {
					continue
				}
				if tok == DOT || lit == "." {
					tok, lit = u.ScanSkipWhitespace()
				}
				u.tagID = lit

				tok, lit = u.ScanSkipWhitespace()
				if tok == EQUALS {
					u.tagTok, u.tagValue = u.ScanSkipWhitespace()
				}

				switch u.tagTok {
				case STRING, QUANTITY, PATH:
					u.tagTyp = "string"
				case INT:
					u.tagTyp = "int"
				case DOUBLE:
					u.tagTyp = "double"
				}

				f := iv.Field(i)
				err := u.field(f)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}

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
				case STRING, QUANTITY, PATH:
					u.tagTyp = "string"
				case INT:
					u.tagTyp = "int"
				case DOUBLE:
					u.tagTyp = "double"
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
		}
	}
}

func Unmarshal(b []byte, v interface{}) *pureError {
	u := &unmarshaler{
		Scanner: newScanner(b),
	}
	u.unmarshal(v)
	if len(u.errors) > 0 {
		return u.errors[0]
	}
	return nil //u.unmarshal(v)
}

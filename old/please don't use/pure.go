package pure

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
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
	tagExt   string
}

func (u *unmarshaler) typeCheck(field reflect.Value) {
	if u.tagExt != "unquoted" {
		for {
			tok, lit := u.ScanSkipWhitespace()
			if tok == EQUALS || tok == WHITESPACE {
				continue
			}
			u.tagValue = lit
			break
		}
	}

	switch {
	case field.Kind() == reflect.Int:
		_i, err := strconv.Atoi(u.tagValue)
		if err != nil {
			fmt.Println(fmt.Sprintf("bad number value '%s'", u.tagValue))
			return
		}
		field.SetInt(int64(_i))
		return
	case field.Kind() == reflect.String:
		field.SetString(u.tagValue)
		return
	case field.Kind() == reflect.Float64:
		f, err := strconv.ParseFloat(u.tagValue, 64)
		if err != nil {
			fmt.Println(u.newError(fmt.Sprintf("bad floating point value '%s'", u.tagValue)).Error())
			return
		}
		field.SetFloat(f)
		return
	case field.Kind() == reflect.Bool:
		b, err := strconv.ParseBool(u.tagValue)
		if err != nil {
			fmt.Println(u.newError(fmt.Sprintf("bad bool value '%s'", u.tagValue)).Error())
			return
		}
		field.SetBool(b)
		return
	case field.Kind() == reflect.Ptr:
		err := u.group(field.Interface())
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		return
	default:
		fi := field.Interface()
		switch fi.(type) {
		case *Quantity:
			fi = NewQuantity(u.tagValue)
			field.Set(reflect.ValueOf(fi))
			return
		case Quantity:
			fi = NewQuantity(u.tagValue)
			field.Set(indirect(reflect.ValueOf(fi)))
			return
		case *Env:
			fi = NewEnv(u.tagValue)
			field.Set(reflect.ValueOf(fi))
			return
		case Env:
			fi = NewEnv(u.tagValue)
			field.Set(indirect(reflect.ValueOf(fi)))
			return
		case *Path:
			fi = NewPath(u.tagValue)
			field.Set(reflect.ValueOf(fi))
			return
		case Path:
			fi = NewPath(u.tagValue)
			field.Set(indirect(reflect.ValueOf(fi)))
			return
		}

	}
}

func (u *unmarshaler) typeCheckRef(field reflect.Value) {
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
			u.tagValue = fi.(*Quantity).Value_
		case Quantity:
			u.tagTyp = "quantity"
			u.tagValue = fi.(Quantity).Value_
		case *Env:
			u.tagTyp = "env"
			u.tagValue = fi.(*Env).Value
		case Env:
			u.tagTyp = "env"
			u.tagValue = fi.(Env).Value
		case *Path:
			u.tagTyp = "path"
			u.tagValue = fi.(*Path).Value
		case Path:
			u.tagTyp = "Path"
			u.tagValue = fi.(Path).Value
		}
	}
}

func (u *unmarshaler) setTyp() {
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
	var tags []string
	var temp string
	var tok Token
	var lit string
	switch v.Kind() {
	case reflect.Ptr:
		iv := indirect(v.Elem())
		for i := 0; i < iv.NumField(); i++ {
			tag := iv.Type().Field(i).Tag.Get(tagName)

			if tag != "" && tag != "-" && tag == u.tagID {
				field = iv.Field(i)
				// Consume the '=' and get the token and value for the property
				_, u.tagID = u.ScanSkipWhitespace()
				tok, lit = u.ScanSkipWhitespace()
				// type check
				//u.setTyp()
				if tok == REF {

					var field reflect.Value
					_, lit = u.ScanSkipWhitespace()
					temp = lit
					// If the peeked token is a '.' then we're going into a group
					// so lit MUST be a group id (ex. 'someGroupId')
					if b := u.Peek(1); b[0] == '.' {
						group := lit

						//Consime the '.'
						u.ScanSkipWhitespace()

						// Get the property id
						tok, u.tagID = u.ScanSkipWhitespace()

						// Get the struct with the correct tag id from the interface
						struc := u.GetStruct(group, v)

						// reset the tag id from the temp value
						u.tagID = temp

						// get the field inside the struct we just got, with the tag id
						field = u.GetField(lit, struc)
					} else if tok == ARRAY {
						err := u.array(field)
						if err != nil {
							fmt.Println(u.newError(err.Error()).Error())
						}
						continue
					} else {

						// If there's no '.'
						// assume it's a regular property and not a group property
						tok, lit = u.ScanSkipWhitespace()
						field = u.GetField(temp, indirect(reflect.ValueOf(v)))

					}
					u.typeCheckRef(field)
				} else if tok == EQUALS {
					if u.tagExt == "unquoted" {
						u.tagTok, u.tagValue = u.Scanner.ScanUnquotedString()
					} else {
						u.tagTok, u.tagValue = u.ScanSkipWhitespace()
					}
					u.setTyp()
				} else {

					if u.tagExt == "unquoted" {
						tok, lit = u.Scanner.ScanUnquotedString()
					} else {
						tok, lit = u.ScanSkipWhitespace()
					}
					u.setTyp()
				}
				break
			} else if tags = strings.Split(tag, ","); len(tags) > 1 && tags[0] == u.tagID {
				field = iv.Field(i)
				u.tagExt = tags[1]

				if u.tagExt == "unquoted" {
					u.tagTok, u.tagValue = u.Scanner.ScanUnquotedString()
					u.setTyp()
					//u.tagExt = ""
				} else {
					// Consume the '=' and get the token and value for the property
					u.tagTok, u.tagValue = u.ScanSkipWhitespace()

					// type check
					u.setTyp()
				}
				break
			}
		}
	case reflect.Struct:
		iv := indirect(v)
		tv := reflect.TypeOf(v.Interface())

		for i := 0; i < iv.NumField(); i++ {
			tag := tv.Field(i).Tag.Get(tagName)
			if tag != "" && tag != "-" && tag == u.tagID {
				field = iv.Field(i)
				break
			} else if tags = strings.Split(tag, ","); len(tags) > 1 && tags[0] == u.tagID {
				field = iv.Field(i)
				u.tagExt = tags[1]

				if u.tagExt == "unquoted" {
					u.tagTok, u.tagValue = u.Scanner.ScanUnquotedString()
					u.setTyp()
					//u.tagExt = ""
				} else {
					// Consume the '=' and get the token and value for the property
					u.tagTok, u.tagValue = u.ScanSkipWhitespace()

					// type check
					u.setTyp()
				}
			}
		}
	}

	if !field.IsValid() {
		field = indirect(v)
	}
	// There has to be a better way for this
	u.typeCheck(field)
	return nil
}

// Peek copies the unmarshalers buffer (to not advance the buffer we're reading form)
// and returns the next n bytes
func (u *unmarshaler) Peek(n int) []byte {
	return bytes.NewBuffer(u.Scanner.buf.Bytes()).Next(n)
}

// This is not pretty, but it works ¯\_(ツ)_/¯
func (u *unmarshaler) PeekLiteral() string {
	var s string
	temp := u.Scanner.buf
	buf := u.Scanner.buf.Bytes()
	u.Scanner.buf = temp
	for i, b := range buf {
		//b, err := buf.ReadByte()

		if IsAlpha(b) {
			s += string(b)
			for _, by := range buf[i+1:] {

				if IsWhitespace(by) {
					break
				}
				s += string(by)
			}
			break
		}
	}
	return s
}

func (u *unmarshaler) group(v interface{}) *pureError {
	iv := indirect(reflect.ValueOf(v))
	tv := reflect.TypeOf(v)
	var tags []string
	// var temp string
	for i := 0; i < iv.NumField(); i++ {
		tag := tv.Elem().Field(i).Tag.Get(tagName)
		if tag == u.tagID {
			if tags = strings.Split(tag, ","); len(tags) > 1 {
				u.tagExt = tags[1]
			}
			f := iv.Field(i)
			for {
				tok, lit := u.Scan()
				if tok == EOF || tok == RBRACK {
					return nil
				}

				if lit == "\r" {
					if b := u.Peek(2); len(b) > 0 && b[0] == '\n' && (IsWhitespace(b[len(b)-1])) {
						u.tagTok, u.tagID = u.ScanSkipWhitespace()
						break
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
	iv := indirect(reflect.ValueOf(v))
	for i := 0; i < iv.NumField(); i++ {
		tag := reflect.TypeOf(v).Elem().Field(i).Tag.Get(tagName)
		if tag == name {
			return iv.Field(i)
		}
	}
	return reflect.ValueOf(v)
}

func (u *unmarshaler) GetField(name string, v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		iv := indirect(v.Elem())

		for i := 0; i < iv.NumField(); i++ {
			tag := iv.Type().Field(i).Tag.Get(tagName)

			if tag == name {
				return iv.Field(i)
			}
		}
	}

	if v.Kind() == reflect.Struct {
		iv := indirect(v)
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

func (u *unmarshaler) appendArray(field reflect.Value) reflect.Value {
	switch u.tagTyp {
	case "string":
		return reflect.Append(field, reflect.ValueOf(u.tagValue))
	case "int":
		_i, err := strconv.Atoi(u.tagValue)
		if err != nil {
			fmt.Printf("invalid integer value '%s'", u.tagValue)
			return reflect.Zero(nil)
		}
		return reflect.Append(field, reflect.ValueOf(_i))
	case "double":
		d, err := strconv.ParseFloat(u.tagValue, 64)
		if err != nil {
			fmt.Printf("invalid floating point value '%s'", u.tagValue)
			return reflect.Zero(nil)
		}
		return reflect.Append(field, reflect.ValueOf(d))
	case "bool":
		b, err := strconv.ParseBool(u.tagValue)
		if err != nil {
			fmt.Printf("invalid boolean value '%s'", u.tagValue)
			return reflect.Zero(nil)
		}
		return reflect.Append(field, reflect.ValueOf(b))
	case "path":
		p := NewPath(u.tagValue)
		return reflect.Append(field, reflect.ValueOf(&p))
	case "quantity":
		q := NewQuantity(u.tagValue)
		return reflect.Append(field, reflect.ValueOf(&q))
	case "env":
		e := NewEnv(u.tagValue)
		return reflect.Append(field, reflect.ValueOf(&e))
	}
	return reflect.Zero(reflect.TypeOf(nil))
}

func (u *unmarshaler) pmap(name string, v reflect.Value) *pureError {
	var field reflect.Value
	var _i int
	var tags []string
	for i := 0; i < v.NumField(); i++ {
		tag := v.Type().Field(i).Tag.Get(tagName)
		if tag == name {
			if tags = strings.Split(tag, ","); len(tags) > 1 {
				u.tagExt = tags[1]
			}
			field = v.Field(i)
			_i = i
			break
		}
		field = v
	}

	if u.tagTok == IDENTIFIER {
		key := u.tagID
		if tok, _ := u.ScanSkipWhitespace(); tok == EQUALS {
			if u.tagExt == "unquoted" {
				u.tagTok, u.tagValue = u.Scanner.ScanUnquotedString()
			} else {
				u.tagTok, u.tagValue = u.ScanSkipWhitespace()
			}
			u.setTyp()
			switch u.tagTyp {
			case "int":
				ii, err := strconv.Atoi(u.tagValue)
				if err != nil {
					fmt.Println(err.Error())
					return nil
				}
				field.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(ii))
			case "double":
				d, err := strconv.ParseFloat(u.tagValue, 64)
				if err != nil {
					fmt.Println(err.Error())
					return nil
				}
				field.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(d))
			case "string":
				field.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(u.tagValue))
			case "bool":
				b, err := strconv.ParseBool(u.tagValue)
				if err != nil {
					fmt.Println(err.Error())
					return nil
				}
				field.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(b))
			case "path":
				p := NewPath(u.tagValue)
				field.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(&p))
			case "quantity":
				q := NewQuantity(u.tagValue)
				field.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(&q))
			case "env":
				e := NewEnv(u.tagValue)
				field.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(&e))
			}
			v.Field(_i).Set(field)
		}
	}

	if u.tagTok == GROUP {
		key := u.tagID
		u.tagTok, u.tagID = u.ScanSkipWhitespace()
		val := reflect.New(reflect.TypeOf(field.Interface()).Elem()).Interface()
		u.group(val)

		field.SetMapIndex(reflect.ValueOf(key), indirect(reflect.ValueOf(val)))
		v.Field(_i).Set(field)
	}
	return nil
}

func (u *unmarshaler) array(v reflect.Value) *pureError {
	var field reflect.Value
	iv := indirect(v)
	temp := u.tagID
	var tags []string
	for i := 0; i < iv.NumField(); i++ {
		tag := iv.Type().Field(i).Tag.Get(tagName)
		if tag == u.tagID {
			field = iv.Field(i)
			if tags = strings.Split(tag, ","); len(tags) > 1 {
				u.tagExt = tags[1]
			}
			break
		}
	}

	for {
		tok, lit := u.ScanSkipWhitespace()

		if tok == EOF || tok == RBRACK {
			return nil
		}

		if lit == "[" {
			for {
				if u.tagExt == "unquoted" {
					u.tagTok, u.tagValue = u.Scanner.ScanUnquotedString()
				} else {
					tok, lit = u.ScanSkipWhitespace()
				}
				if tok == RBRACK || tok == EOF {
					return nil
				}
				if tok == GROUP || tok == IDENTIFIER {
					u.tagTok = tok
					u.tagID = lit
					u.pmap(temp, v)
					continue
				}
				break
			}
			u.tagTok, u.tagValue = tok, lit
			u.setTyp()
		} else {
			u.tagValue = lit
			u.setTyp()
		}

		switch field.Kind() {
		case reflect.Slice, reflect.Array:
			field.Set(u.appendArray(field))
		}
	}
}

// Gotta pretty this up it's really ugly
// Makes me wanna vomit
func (u *unmarshaler) unmarshal(v interface{}) {
	pv := indirect(reflect.ValueOf(v))
	for {
		u.tagExt = ""
		u.tagTok, u.tagID = u.ScanSkipWhitespace()

		if u.tagTok == EOF {
			return
		}

		switch u.tagTok {
		case IDENTIFIER:

			// Check if the next token is a an '='
			if tok, _ := u.ScanSkipWhitespace(); tok == EQUALS {

				if b := u.Peek(2); b[1] == '[' || b[0] == '[' {
					err := u.array(pv)
					if err != nil {
						fmt.Println(u.newError(err.Error()).Error())
					}
					continue
				}

				// Else if it's a reference
			} else if tok == REF {
				var field reflect.Value

				// Store the token id in temp
				// and get the next token
				temp := u.tagID
				tok, lit := u.ScanSkipWhitespace()

				// If the peeked token is a '.' then we're going into a group
				// so lit MUST be a group id (ex. 'someGroupId')
				if b := u.Peek(1); b[0] == '.' {
					group := lit

					//Consime the '.'
					u.ScanSkipWhitespace()

					// Get the property id
					tok, u.tagID = u.ScanSkipWhitespace()

					// Get the struct with the correct tag id from the interface
					struc := u.GetStruct(group, v)

					// reset the tag id from the temp value
					u.tagID = temp

					// get the field inside the struct we just got, with the tag id
					field = u.GetField(lit, struc)
				} else if tok == ARRAY {
					err := u.array(field)
					if err != nil {
						fmt.Println(u.newError(err.Error()).Error())
					}
					continue
				} else {

					// If there's no '.'
					// assume it's a regular property and not a group property
					tok, lit = u.ScanSkipWhitespace()
					field = u.GetField(u.tagID, indirect(reflect.ValueOf(v)))
				}

				// type check
				// this can probably be made prettier
				u.typeCheckRef(field)
			}

			// assign the value to the field
			err := u.field(pv)
			if err != nil {
				fmt.Println(u.newError(err.Error()).Error())
			}

		case GROUP:
			err := u.group(v)
			if err != nil {
				fmt.Println(u.newError(err.Error()).Error())
			}
		case INCLUDE:

			// if we're including a file all we do is unmarshal that BEFORE we do anything else
			b, err := ioutil.ReadFile(u.tagID)
			if err != nil {
				fmt.Println(u.newError(err.Error()).Error())
			}
			Unmarshal(b, v)
		}
	}
}

func Unmarshal(b []byte, v interface{}) {
	u := &unmarshaler{
		Scanner: newScanner(b),
	}
	u.unmarshal(v)
}

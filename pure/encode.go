package pure

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

type encoder struct {
	buf         *bytes.Buffer
	indentSize  int
	indentlevel int
}

func (e *encoder) group(v reflect.Value) {
	var iv reflect.Value
	if v.Kind() == reflect.Ptr {
		iv = indirect(v.Elem())
	} else if v.Kind() == reflect.Struct {
		iv = indirect(v)
	}

	for i := 0; i < iv.NumField(); i++ {
		e.buf.WriteString("\r\n")
		tag := iv.Type().Field(i).Tag.Get("pure")

		if split := strings.Split(tag, ","); len(split) > 1 {
			tag = split[0]
		}

		if tag != "" && tag != "-" {
			field := iv.Field(i)
			for j := 0; j < e.indentSize*e.indentlevel; j++ {
				e.buf.WriteByte(' ')
			}

			switch field.Kind() {
			case reflect.Int, reflect.Float64, reflect.Bool:
				e.buf.WriteString(fmt.Sprintf("%s = %v", tag, field))
			case reflect.String:
				e.buf.WriteString(fmt.Sprintf("%s = \"%v\"", tag, field))
			case reflect.Ptr, reflect.Struct:
				e.indentlevel++
				e.buf.WriteString(tag)
				e.group(field)
				e.indentlevel--
			case reflect.Slice:
				e.buf.WriteString(fmt.Sprintf("%s = [", tag))
				e.array(field)
				e.buf.WriteString("\r\n]\r\n")
			case reflect.Map:
				e.buf.WriteString(fmt.Sprintf("%s = [", tag))
				e.keyValuePair(field)
				e.buf.WriteString("\r\n]\r\n")
			}
		}
	}
	e.buf.WriteString("\r\n")
}

func (e *encoder) keyValuePair(v reflect.Value) {
	keys := v.MapKeys()
	for i := 0; i < v.Len(); i++ {
		e.buf.WriteString("\r\n")
		for i := 0; i < e.indentSize*e.indentlevel; i++ {
			e.buf.WriteByte(' ')
		}

		key := keys[i]
		val := v.MapIndex(key)
		switch reflect.TypeOf(v.Interface()).Elem().Kind() {
		case reflect.Int, reflect.Float64, reflect.Bool:
			e.buf.WriteString(fmt.Sprintf("%v = %v", key, val))
		case reflect.String:
			e.buf.WriteString(fmt.Sprintf("%v = \"%v\"", key, val))
		case reflect.Ptr, reflect.Struct:
			e.indentlevel++
			e.buf.WriteString(fmt.Sprintf("%v", key))
			e.group(val)
			e.indentlevel--
		case reflect.Slice:
			e.indentlevel++
			e.buf.WriteString(fmt.Sprintf("%v = [", key))
			e.array(val)
			e.buf.WriteString("\n]\n")
			e.indentlevel--
		case reflect.Map:
			e.indentlevel++
			e.buf.WriteString(fmt.Sprintf("%v = [", key))
			e.keyValuePair(val)
			e.buf.WriteString("\n]\n")
			e.indentlevel--
		}
	}
}

func (e *encoder) array(v reflect.Value) {

	for i := 0; i < v.Len(); i++ {
		e.buf.WriteString("\r\n")
		for i := 0; i < e.indentSize*e.indentlevel; i++ {
			e.buf.WriteByte(' ')
		}

		switch reflect.TypeOf(v.Interface()).Elem().Kind() {
		case reflect.Int, reflect.Float64, reflect.Bool:
			e.buf.WriteString(fmt.Sprintf("%v", v.Index(i)))
		case reflect.String:
			e.buf.WriteString(fmt.Sprintf("\"%v\"", v.Index(i)))
		}
	}
}

func (e *encoder) marshal(v interface{}) error {
	iv := indirect(reflect.ValueOf(v))
	var noQuotes bool
	for i := 0; i < iv.NumField(); i++ {
		tag := iv.Type().Field(i).Tag.Get("pure")

		if split := strings.Split(tag, ","); len(split) > 1 {
			tag = split[0]
			if split[1] == "quantity" || split[1] == "path" || split[1] == "env" || split[1] == "unquoted" {
				noQuotes = true
			}
		}
		if tag != "" && tag != "-" {
			field := iv.Field(i)

			switch field.Kind() {
			case reflect.Int, reflect.Float64, reflect.Bool:
				e.buf.WriteString(fmt.Sprintf("%s = %v\n", tag, field))
			case reflect.String:
				if noQuotes {
					e.buf.WriteString(fmt.Sprintf("%s = %v\n", tag, field))
					noQuotes = false
				} else {
					e.buf.WriteString(fmt.Sprintf("%s = \"%v\"\n", tag, field))
				}
			case reflect.Ptr, reflect.Struct:
				e.buf.WriteString(tag)
				e.group(field)
			case reflect.Slice:
				e.buf.WriteString(fmt.Sprintf("%s = [", tag))
				e.array(field)
				e.buf.WriteString("\r\n]\r\n")
			case reflect.Map:
				e.buf.WriteString(fmt.Sprintf("%s = [", tag))
				e.keyValuePair(field)
				e.buf.WriteString("\r\n]\r\n")
			}
		}
	}

	return nil
}

func Marhsal(v interface{}) ([]byte, error) {
	e := &encoder{
		buf:         &bytes.Buffer{},
		indentSize:  4,
		indentlevel: 1,
	}
	err := e.marshal(v)
	if err != nil {
		return nil, err
	}
	return e.buf.Bytes(), nil
}

package pure

import (
	"bytes"
	"fmt"
	"reflect"
)

type encoder struct {
	buf         *bytes.Buffer
	indent      int
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
		tag := iv.Type().Field(i).Tag.Get(tagName)

		if tag != "" && tag != "-" {
			field := iv.Field(i)
			for j := 0; j < e.indent*e.indentlevel; j++ {
				e.buf.WriteByte(' ')
			}

			if fi := field.Interface(); fi != nil {
				switch fi.(type) {
				case *Quantity:
					e.buf.WriteString(fmt.Sprintf("%s = %s", tag, fi.(*Quantity).value))
					continue
				case Quantity:
					e.buf.WriteString(fmt.Sprintf("%s = %s", tag, fi.(Quantity).value))
					continue
				case *Path:
					e.buf.WriteString(fmt.Sprintf("%s = %s", tag, fi.(*Path).value))
					continue
				case Path:
					e.buf.WriteString(fmt.Sprintf("%s = %s", tag, fi.(Path).value))
					continue
				case *Env:
					e.buf.WriteString(fmt.Sprintf("%s = %s", tag, fi.(*Env).value))
					continue
				case Env:
					e.buf.WriteString(fmt.Sprintf("%s = %s", tag, fi.(Env).value))
					continue
				}
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
				e.mäp(field)
				e.buf.WriteString("\r\n]\\n")
			}
		}
	}
	e.buf.WriteString("\r\n")
}

func (e *encoder) mäp(v reflect.Value) {
	for i := 0; i < v.Len(); i++ {
		e.buf.WriteString("\r\n")
		for i := 0; i < e.indent*e.indentlevel; i++ {
			e.buf.WriteByte(' ')
		}

		if fi := v.Interface(); fi != nil {
			switch fi.(type) {
			case *Quantity:
				e.buf.WriteString(fmt.Sprintf("%v = %v", v.MapKeys()[i], fi.(*Quantity).value))
				continue
			case Quantity:
				e.buf.WriteString(fmt.Sprintf("%v = %v", v.MapKeys()[i], fi.(Quantity).value))
				continue
			case *Path:
				e.buf.WriteString(fmt.Sprintf("%v = %v", v.MapKeys()[i], fi.(*Path).value))
				continue
			case Path:
				e.buf.WriteString(fmt.Sprintf("%v = %v", v.MapKeys()[i], fi.(Path).value))
				continue
			case *Env:
				e.buf.WriteString(fmt.Sprintf("%v = %v", v.MapKeys()[i], fi.(*Env).value))
				continue
			case Env:
				e.buf.WriteString(fmt.Sprintf("%v = %v", v.MapKeys()[i], fi.(Env).value))
				continue
			}
		}

		switch reflect.TypeOf(v.Interface()).Elem().Kind() {
		case reflect.Int, reflect.Float64, reflect.Bool:
			e.buf.WriteString(fmt.Sprintf("%v = %v", v.MapKeys()[i], v.MapIndex(v.MapKeys()[i])))
		case reflect.String:
			e.buf.WriteString(fmt.Sprintf("%v = \"%v\"", v.MapKeys()[i], v.MapIndex(v.MapKeys()[i])))
		case reflect.Ptr, reflect.Struct:
			e.indentlevel++
			e.buf.WriteString(fmt.Sprintf("%v", v.MapKeys()[i]))
			e.group(v.MapIndex(v.MapKeys()[i]))
			e.indentlevel--
		case reflect.Slice:
			e.indentlevel++
			e.buf.WriteString(fmt.Sprintf("%v = [", v.MapKeys()[i]))
			e.array(v.MapIndex(v.MapKeys()[i]))
			e.buf.WriteString("\n]\n")
			e.indentlevel--
		case reflect.Map:
			e.indentlevel++
			e.buf.WriteString(fmt.Sprintf("%v = [", v.MapKeys()[i]))
			e.mäp(v.MapIndex(v.MapKeys()[i]))
			e.buf.WriteString("\n]\n")
			e.indentlevel--
		}
	}
}

func (e *encoder) array(v reflect.Value) {

	for i := 0; i < v.Len(); i++ {
		e.buf.WriteString("\r\n")
		for i := 0; i < e.indent*e.indentlevel; i++ {
			e.buf.WriteByte(' ')
		}

		if fi := v.Interface(); fi != nil {
			switch fi.(type) {
			case *Quantity:
				e.buf.WriteString(fmt.Sprintf("%s", fi.(*Quantity).value))
				continue
			case Quantity:
				e.buf.WriteString(fmt.Sprintf("%s", fi.(Quantity).value))
				continue
			case *Path:
				e.buf.WriteString(fmt.Sprintf("%s", fi.(*Path).value))
				continue
			case Path:
				e.buf.WriteString(fmt.Sprintf("%s", fi.(Path).value))
				continue
			case *Env:
				e.buf.WriteString(fmt.Sprintf("%s", fi.(*Env).value))
				continue
			case Env:
				e.buf.WriteString(fmt.Sprintf("%s", fi.(Env).value))
				continue
			}
		}

		switch reflect.TypeOf(v.Interface()).Elem().Kind() {
		case reflect.Int, reflect.Float64, reflect.Bool:
			e.buf.WriteString(fmt.Sprintf("%v", v.Index(i)))
		case reflect.String:
			e.buf.WriteString(fmt.Sprintf("\"%v\"", v.Index(i)))
		}
	}
}

func (e *encoder) marshal(v interface{}) *pureError {
	iv := indirect(reflect.ValueOf(v))

	for i := 0; i < iv.NumField(); i++ {
		tag := iv.Type().Field(i).Tag.Get(tagName)

		if tag != "" && tag != "-" {
			field := iv.Field(i)

			if fi := field.Interface(); fi != nil {
				switch fi.(type) {
				case *Quantity:
					e.buf.WriteString(fmt.Sprintf("%s = %s\n", tag, fi.(*Quantity).value))
					continue
				case Quantity:
					e.buf.WriteString(fmt.Sprintf("%s = %s\n", tag, fi.(Quantity).value))
					continue
				case *Path:
					e.buf.WriteString(fmt.Sprintf("%s = %s\n", tag, fi.(*Path).value))
					continue
				case Path:
					e.buf.WriteString(fmt.Sprintf("%s = %s\n", tag, fi.(Path).value))
					continue
				case *Env:
					e.buf.WriteString(fmt.Sprintf("%s = %s\n", tag, fi.(*Env).value))
					continue
				case Env:
					e.buf.WriteString(fmt.Sprintf("%s = %s\n", tag, fi.(Env).value))
					continue
				}
			}

			switch field.Kind() {
			case reflect.Int, reflect.Float64, reflect.Bool:
				e.buf.WriteString(fmt.Sprintf("%s = %v\n", tag, field))
			case reflect.String:
				e.buf.WriteString(fmt.Sprintf("%s = \"%v\"\n", tag, field))
			case reflect.Ptr, reflect.Struct:
				e.buf.WriteString(tag)
				e.group(field)
			case reflect.Slice:
				e.buf.WriteString(fmt.Sprintf("%s = [", tag))
				e.array(field)
				e.buf.WriteString("\r\n]\r\n")
			case reflect.Map:
				e.buf.WriteString(fmt.Sprintf("%s = [", tag))
				e.mäp(field)
				e.buf.WriteString("\r\n]\r\n")
			}
		}
	}

	return nil
}

func Marhsal(v interface{}) ([]byte, *pureError) {
	e := &encoder{
		buf:         &bytes.Buffer{},
		indent:      4,
		indentlevel: 1,
	}
	err := e.marshal(v)
	if err != nil {
		return nil, err
	}
	return e.buf.Bytes(), nil
}

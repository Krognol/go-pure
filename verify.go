package pure

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type Verifier struct {
	buf *bytes.Buffer
	u   *unmarshaler
}

func NewVerifier() *Verifier {
	v := &Verifier{
		u:   &unmarshaler{},
		buf: &bytes.Buffer{},
	}
	return v
}

func (v *Verifier) Verify(i interface{}) error {
	f, err := os.Create("./verification.json")
	if err != nil {
		return err
	}
	arr, err := v.verify(i)
	if err != nil {
		return err
	}
	_, err = f.WriteString("{\r\n" + strings.Join(arr, ",\n") + "\r\n}")
	if err != nil {
		return err
	}
	return nil
}

func (v *Verifier) verify(i interface{}) ([]string, error) {
	iv := v.u.indirect(reflect.ValueOf(i))
	var arr []string

	if iv.Kind() == reflect.Ptr {
		piv := v.u.indirect(iv.Elem())
		for i := 0; i < piv.NumField(); i++ {
			tag := piv.Type().Field(i).Tag.Get("pure")
			if tag != "" && tag != "-" {
				f := iv.Field(i)
				switch f.Kind() {
				case reflect.Int:
					arr = append(arr, fmt.Sprintf("    \"%s\": %d", tag, f.Int()))
				case reflect.Float64:
					arr = append(arr, fmt.Sprintf("    \"%s\": %f", tag, f.Float()))
				case reflect.String:
					arr = append(arr, fmt.Sprintf("    \"%s\": \"%s\"", tag, f.String()))
				case reflect.Bool:
					arr = append(arr, fmt.Sprintf("    \"%s\": %t", tag, f.Bool()))
				case reflect.Struct:
					ar, err := v.verify(f)
					ar[0] = "{\n    " + ar[0]
					ar[len(ar)-1] = ar[1] + "\n    }"
					if err != nil {
						return nil, err
					}
					arr = append(arr, ar...)
				case reflect.Ptr:
					ar, err := v.verify(f.Interface())
					ar[0] = "{\n    " + ar[0]
					ar[len(ar)-1] = ar[1] + "\n    }"
					if err != nil {
						return nil, err
					}
					arr = append(arr, ar...)
				}
			}
		}
	}

	if iv.Kind() == reflect.Struct {
		tv := reflect.TypeOf(iv.Interface())
		for i := 0; i < iv.NumField(); i++ {
			tag := tv.Field(i).Tag.Get("pure")
			if tag != "" && tag != "-" {
				f := iv.Field(i)
				switch f.Kind() {
				case reflect.Int:
					arr = append(arr, fmt.Sprintf("    \"%s\": %d", tag, f.Int()))
				case reflect.Float64:
					arr = append(arr, fmt.Sprintf("    \"%s\": %F", tag, f.Float()))
				case reflect.String:
					arr = append(arr, fmt.Sprintf("    \"%s\": \"%s\"", tag, f.String()))
				case reflect.Bool:
					arr = append(arr, fmt.Sprintf("    \"%s\": %t", tag, f.Bool()))
				case reflect.Struct:
					ar, err := v.verify(f)
					ar[0] = "{\n    " + ar[0]
					ar[len(ar)-1] = ar[1] + "\n    }"
					if err != nil {
						return nil, err
					}
					arr = append(arr, ar...)
				case reflect.Ptr:
					ar, err := v.verify(f.Interface())
					ar[0] = "{\n    " + ar[0]
					ar[len(ar)-1] = ar[len(ar)-1] + "\n    }"
					if err != nil {
						return nil, err
					}
					arr = append(arr, fmt.Sprintf("    \"%s\": %s", tag, ar[0]))
				}
			}
		}
	}

	return arr, nil
}

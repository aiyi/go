package db

import (
	"bytes"
	"reflect"

	"github.com/aiyi/go/utils"
)

func SqlUpdateSetArgs(s *bytes.Buffer, para interface{}, args *[]interface{}) int {
	x := 0
	v := reflect.Indirect(reflect.ValueOf(para))

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() != reflect.Ptr && field.Kind() != reflect.Slice {
			continue
		}

		if field.IsNil() == false {
			if x > 0 {
				s.WriteString(", ")
			}
			s.WriteString(utils.ToFieldName(v.Type().Field(i).Name))
			s.WriteString("=?")
			if field.Kind() == reflect.Ptr {
				*args = append(*args, field.Elem().Interface())
			} else {
				*args = append(*args, field.Slice(0, field.Len()).Interface())
			}
			x++
		}
	}

	return x
}

package db

import (
	"bytes"
	"reflect"
	"time"
)

func SqlUpdateSetArgs(s *bytes.Buffer, para interface{}, args *[]interface{}) int {
	x := 0
	v := reflect.Indirect(reflect.ValueOf(para))

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() != reflect.Ptr && field.Kind() != reflect.Slice {
			continue
		}

		key := v.Type().Field(i).Tag.Get("json")

		if field.IsNil() == false || key == "modified" {
			if x > 0 {
				s.WriteString(", ")
			}

			s.WriteString(key)
			s.WriteString("=?")
			if key == "modified" {
				*args = append(*args, time.Now().Unix())
			} else {
				if field.Kind() == reflect.Ptr {
					*args = append(*args, field.Elem().Interface())
				} else {
					*args = append(*args, field.Slice(0, field.Len()).Interface())
				}
			}

			x++
		}
	}

	s.WriteString(" ")

	return x
}

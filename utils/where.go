package utils

import (
	"reflect"
//	"reflect"
	"errors"
	"fmt"
	"strings"
	"encoding/json"
	
	"github.com/antonholmquist/jason"
)

// where = {key1: [{op: val}, {op: val}], key2: [{op: val}, {op: val}], ...}
// where = {or: [{key1: [{op: val}, {op: val}, ...], ...}, {key1: [{op: val}, {op: val}], ...}]}
type Expression struct {
	op string
	value interface{}
}
type Condition map[string]*[]Expression
type Where []*Condition
type Filter struct {
     where Where
}

var opMap map[string]string

func valString(v interface{}) (s string) {
	switch v.(type) {
	case []interface{}:
		a, _ := v.([]interface{})
		for _, ai := range a {
			s = valString(ai)
		}
	case bool:
		b, _ := v.(bool)
		if b == true {
			s = "TRUE"
		} else {
			s = "FALSE"
		}
	case string:
		s, _ = v.(string)
	case json.Number:
		n, _ := v.(json.Number)
		s = n.String()
	default:
		fmt.Println("unknown type? don't know how to convert")
		s = ""
	}
	
	return s
}

func opString(op string) (string) {
	switch op {
	case "eq":
		return "="
	case "lt":
		return "<"
	case "lte":
		return "<="
	case "gt":
		return ">"
	case "gte":
		return ">="
	case "ne":
		return "<>"
	case "in":
		return "IN"
	case "nin":
		return "NOT IN"
	default:
		fmt.Println(op, " not found")
		return ""
	}
}

func (w *Where) SqlString() string {
	// condition number
	cn := len(*w)

	s := "where "
	for i, c := range *w {
		// keyword number
		kn := len(*c)
		ki := 0
		
		s += "("
		for ck, cv := range *c {
			// expression number
			en := len(*cv)
			
			// TODO check whether ck is a filed of struct
			
			s += "("
			for j, ei := range *cv {
				s += ck + " " + opString(ei.op) + " " + valString(ei.value)
				if en > 1 && j < en - 1 {
					s += " AND "
				}
			}
			s += ")"
			
			if kn > 1 && ki < kn -1{
				s += " AND "
			}

			ki += 1
		}
		s += ")"
		
		if cn > 1 && i < cn - 1 {
			s += " OR "
		}
	}
	return s
}

func (w *Where) Values() []interface{} {
	ia := make([]interface{}, 4)

	for _, c := range *w {		
		for _, cv := range *c {		
			for _, ei := range *cv {
				ia = append(ia, valString(ei.value))
			}
		}
	}

	return ia
}

func (w Where) String() string {
	s := ""
	for _, c := range w {
		s += "["
		for ck, cv := range *c {
			s += "{" + ck + ":" + "[";
			for _, ei := range *cv {
				s += "{" + ei.op + ":" + valString(ei.value) + "}"
			}
			s += "]" + "}"
		}
		s += "]"
	}
	return s
}

func parseCondition(v *jason.Object) (c *Condition, err error) {
	conds, err := v.Object()
	if conds == nil {
		return nil, err
	}
	
	c = &Condition{}
		
	for ck, cv := range conds.Map() {		
		ea, _ := cv.Array()
		if (ea == nil) {
			continue
		}
		
		es := make([]Expression, len(ea))
		
		for i, ai := range ea {
			expr, _ := ai.Object()
			for ek, ev := range expr.Map() {						
				e := &Expression{}
				e.op = ek
				
				switch ev.Interface().(type) {
				case []interface{}:
					e.value, _ = ev.Array()
				case bool:
					e.value, _ = ev.Boolean()
				case string:
					e.value, _ = ev.String()
				case json.Number:
					e.value, _ = ev.Number()
				default:
					e.value = ev.Interface()
					fmt.Println(ev, " unsupported type ", 
						reflect.ValueOf(ev.Interface()).Type())
				}
				
				es[i] = *e
			}
		}
				
		(*c)[ck] = &es
	}
	
	return c, nil
}

func (w *Where) ParseWhere(str string) (err error) {
	sa := strings.Split(str, "=")
	if len(sa) != 2 {
		fmt.Println("invalid str")
		return errors.New("invalid query string")
	}
		
	root, err := jason.NewObjectFromBytes([]byte(sa[1]))
	if err != nil {
		fmt.Println("parse json failed")
		return err
	}
		
	oa, err := root.GetObjectArray("or")
	if err != nil {
		fmt.Println("get or failed")
		return err
	}
	if oa == nil {
		c, err := parseCondition(root)
		if err != nil {
			fmt.Println("parseCondition failed")
			return err
		}

		*w = append(*w, c)
	} else {
		for _, o := range oa {
			c, err := parseCondition(o)
			if err != nil {
				fmt.Println("parseCondition failed")
				return err
			}
		
			*w = append(*w, c)
		}
	}
	
	return nil
}

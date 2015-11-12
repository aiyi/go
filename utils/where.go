package utils

import (
	"net/url"
	"reflect"
	"fmt"
	"strings"
	"strconv"
	"encoding/json"
	
	"github.com/antonholmquist/jason"
)

// where = {key1: {op: val, op: val}, key2: {op: val, op: val}, ...}

type Expression struct {
	op string
	value interface{}
}
type Condition map[string]*[]Expression
type Where []*Condition

type Order struct {
	key	string
	asc	bool
}

type Filter struct {
	where			*Where
	orders          []*Order
	skip          	int64
	limit           int64
	unscoped        bool
	SoftDelete      bool
}

func valItem(item interface{}) interface{} {
	switch item.(type) {
	case bool:
		b, _ := item.(bool)
		return b
	case string:
		s, _ := item.(string)
		return s
	case json.Number:
		n, _ := item.(json.Number)
		i, _ := n.Int64()
		return i
	default:
		fmt.Println("valItem parse failed, type: ", reflect.ValueOf(item).Type())
		return nil
	}
}

func valArray(v interface{}) (a []interface{}) {
	switch v.(type) {
	case []*jason.Value:
		va, _ := v.([]*jason.Value)
		for _, item := range va {
			a = append(a, valItem(item.Interface()))
		}
	default:
		fmt.Println("valArray parse failed, type: ", reflect.ValueOf(v).Type())
	}
	
	return a
}

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
		s = "'" + s + "'"
	case json.Number:
		n, _ := v.(json.Number)
		s = n.String()
	case []*jason.Value:
		a, _ := v.([]*jason.Value)
		s = "["
		for i, ai := range a {			
			s += valString(ai.Interface())
			if i < len(a) -1 {
				s +=  ", "
			}
		}
		s += "]"
	default:
		fmt.Println("unknown type? don't know how to convert: ", reflect.ValueOf(v).Type())
		s = ""
	}
	
	return s
}

func opString(op string) (string) {
	switch op {
	case "$eq":
		return "="
	case "$lt":
		return "<"
	case "$lte":
		return "<="
	case "$gt":
		return ">"
	case "$gte":
		return ">="
	case "$ne":
		return "<>"
	case "$in":
		return "IN"
	case "$nin":
		return "NOT IN"
	default:
		fmt.Println(op, " not found")
		return ""
	}
}

func (f *Filter) SqlString() (string, []interface{}) {
	var ia []interface{}
	s := ""
	
	// where sql
	
	if f.where != nil {
		// condition number
		cn := len(*f.where)
	
		for i, conds := range *f.where {
			// keyword number
			kn := len(*conds)
			ki := 0
			
			s += "("
			for ck, cv := range *conds {
				// expression number
				en := len(*cv)
				
				// TODO check whether ck is a filed of struct
				
				s += "("
				for j, exp := range *cv {
					// operation
					if exp.op == "$in" || exp.op == "$nin" {
						va, ok := exp.value.([]*jason.Value)
						if !ok {
							fmt.Println("in value wrong, not a array")
						}
	
						s += ck + " " + opString(exp.op) + " " + "("
						
						for k, ei := range valArray(exp.value) {
							s += "?"
							if k != len(va) - 1 {
								s += ","
							}
							ia = append(ia, ei)
						}
						s += ")"
					} else {
						s += ck + " " + opString(exp.op) + " " + "?"
						ia = append(ia, exp.value)
					}
	
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
	}
	
	// order by sql
	if len(f.orders) > 0 {
		s += " ORDER BY "
		
		for i, order := range f.orders {
			s += order.key
			if order.asc != true {
				s += " DESC"
			}
			if i < len(f.orders) - 1 {
				s += ", "
			}
		}
	}
	
	// limit sql
	if f.limit > 0 {
		s += "LIMIT " + strconv.FormatInt(f.limit, 10)
	}
	// skip sql
	if f.skip > 0 {
		s += "OFFSET " + strconv.FormatInt(f.skip, 10)
	}
	
	return s, ia
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
		fmt.Println("get conds failed")
		return nil, err
	}
	
	c = &Condition{}
			
	for ck, cv := range conds.Map() {
		ea := &[]Expression{}
		
		eo, err := cv.Object()
		if err != nil {
			// cv is value
			e := &Expression{}
			e.op = "$eq"

			switch cv.Interface().(type) {
			case []interface{}:
				e.value, _ = cv.Array()
			case bool:
				e.value, _ = cv.Boolean()
			case string:
				e.value, _ = cv.String()
			case json.Number:
				e.value, _ = cv.Number()
			default:
				e.value = cv.Interface()
				fmt.Println(cv, " unsupported type ", 
					reflect.ValueOf(cv.Interface()).Type())
			}
			
			*ea = append(*ea, *e)
		} else {
			// cv is a object
			i := 0
			for ek, ev := range eo.Map() {
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
				
				*ea = append(*ea, *e)
				i += 1
			}
		}

		(*c)[ck] = ea
	}
	
	return c, nil
}

func (f *Filter) parseWhere(str string) (err error) {
	where := &Where{}
	
	root, err := jason.NewObjectFromBytes([]byte(str))
	if err != nil {
		fmt.Println("parse json failed")
		return err
	}
	
	oa, _ := root.GetObjectArray("$or")
	if oa == nil {
		c, err := parseCondition(root)
		if err != nil {
			fmt.Println("parseCondition failed")
			return err
		}

		*where = append(*where, c)
	} else {
		for _, o := range oa {
			c, err := parseCondition(o)
			if err != nil {
				fmt.Println("parseCondition failed")
				return err
			}
		
			*where = append(*where, c)
		}
	}
	
	f.where = where
	
	return nil
}

func (f *Filter) parseOrder(str string) (err error) {
	if str == "" {
		return nil
	}
	
	oa := strings.Split(str, ",")
	
	orders := []*Order{}
	
	for _, o := range oa {
		if o == "" {
			continue
		}
		
		order := &Order{}
				
		if o[0] == '-' {
			order.asc = false
			order.key = string(o[1:])
		} else {
			order.asc = true
			order.key = o
		}
		
		orders = append(orders, order)
	}
	
	f.orders = orders
	
	return nil
}

func (f *Filter) parseLimit(str string) (err error) {
	f.limit, err = strconv.ParseInt(str, 10, 64)
	
	return err
}

func (f *Filter) parseSkip(str string) (err error) {
	f.skip, err = strconv.ParseInt(str, 10, 64)
	
	return err
}

func ParseFilter(query url.Values) (f *Filter, err error) {
	f = new(Filter)
	
	where := query.Get("where")
	f.parseWhere(where)
	
	order := query.Get("order")
	f.parseOrder(order)
	
	limit := query.Get("limit")
	f.parseLimit(limit)
	
	skip := query.Get("skip")
	f.parseSkip(skip)
	
	return f, nil
}



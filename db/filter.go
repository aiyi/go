package db

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
)

type Filter struct {
	whereConditions []*condition
	orConditions    []*condition
	notConditions   []*condition
	orders          []string
	offset          string
	limit           string
	unscoped        bool
	SoftDelete      bool
	SqlVars         []interface{}
}

type condition struct {
	expr string
	arg  interface{}
}

func (s *Filter) Where(expr string, value interface{}) *Filter {
	s.whereConditions = append(s.whereConditions, &condition{expr: expr, arg: value})
	return s
}

func (s *Filter) Not(expr string, value interface{}) *Filter {
	s.notConditions = append(s.notConditions, &condition{expr: expr, arg: value})
	return s
}

func (s *Filter) Or(expr string, value interface{}) *Filter {
	s.orConditions = append(s.orConditions, &condition{expr: expr, arg: value})
	return s
}

func (s *Filter) Order(value string, reorder ...bool) *Filter {
	if len(reorder) > 0 && reorder[0] {
		if value != "" {
			s.orders = []string{value}
		} else {
			s.orders = []string{}
		}
	} else if value != "" {
		s.orders = append(s.orders, value)
	}
	return s
}

func (s *Filter) Limit(value int) *Filter {
	s.limit = fmt.Sprintf("%v", value)
	return s
}

func (s *Filter) Offset(value int) *Filter {
	s.offset = fmt.Sprintf("%v", value)
	return s
}

func (s *Filter) Unscoped() *Filter {
	s.unscoped = true
	return s
}

func (s *Filter) CombinedConditionSql() string {
	return s.WhereSql() + s.orderSql() + s.limitSql() + s.offsetSql()
}

func (s *Filter) WhereSql() (sql string) {
	var primaryConditions, andConditions, orConditions []string

	if !s.unscoped && s.SoftDelete {
		sql := "(deleted = 0)"
		primaryConditions = append(primaryConditions, sql)
	}

	for _, clause := range s.whereConditions {
		if sql := s.buildWhereCondition(clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	for _, clause := range s.orConditions {
		if sql := s.buildWhereCondition(clause); sql != "" {
			orConditions = append(orConditions, sql)
		}
	}

	for _, clause := range s.notConditions {
		if sql := s.buildNotCondition(clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	orSql := strings.Join(orConditions, " OR ")
	combinedSql := strings.Join(andConditions, " AND ")
	if len(combinedSql) > 0 {
		if len(orSql) > 0 {
			combinedSql = combinedSql + " OR " + orSql
		}
	} else {
		combinedSql = orSql
	}

	if len(primaryConditions) > 0 {
		sql = "WHERE " + strings.Join(primaryConditions, " AND ")
		if len(combinedSql) > 0 {
			sql = sql + " AND (" + combinedSql + ")"
		}
	} else if len(combinedSql) > 0 {
		sql = "WHERE " + combinedSql
	}
	return
}

func (s *Filter) orderSql() string {
	if len(s.orders) == 0 {
		return ""
	}
	return " ORDER BY " + strings.Join(s.orders, ",")
}

func (s *Filter) limitSql() string {
	if len(s.limit) == 0 {
		return ""
	}
	return " LIMIT " + s.limit
}

func (s *Filter) offsetSql() string {
	if len(s.offset) == 0 {
		return ""
	}
	return " OFFSET " + s.offset
}

func (s *Filter) AddToVars(value interface{}) {
	s.SqlVars = append(s.SqlVars, value)
}

func (s *Filter) Quote(key string) string {
	return fmt.Sprintf(`"%s"`, key)
}

func (s *Filter) buildWhereCondition(clause *condition) (str string) {
	str = fmt.Sprintf("(%v)", clause.expr)
	arg := clause.arg

	switch reflect.ValueOf(arg).Kind() {
	case reflect.Slice: // For where("id in (?)", []int64{1,2})
		s.AddToVars(arg)
	default:
		if valuer, ok := interface{}(arg).(driver.Valuer); ok {
			arg, _ = valuer.Value()
		}
		s.AddToVars(arg)
	}

	return
}

func (s *Filter) buildNotCondition(clause *condition) (str string) {
	str = fmt.Sprintf("(%v NOT IN (?))", s.Quote(clause.expr))
	s.AddToVars(clause.arg)
	return
}

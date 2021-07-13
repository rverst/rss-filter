package goql

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Conditions interface {
	String() string
	Add(thing *Condition)
	Equals(thing Conditions) bool
	AddDateFormat(format string)
	CheckMap(m map[string]interface{}) (bool, error)
	CheckStruct(s interface{}) (bool, error)
	Conditions() []*Condition
}

type Condition struct {
	Link       Token
	Negate     bool
	Key        string
	Expression string
	ExprType   Token
	Operator   Token
}

type cons struct {
	things      []*Condition
	dateFormats []string
}

func NewConditions() Conditions {
	return &cons{
		dateFormats: []string{time.RFC3339},
	}
}

func (t cons) Conditions() []*Condition {
	return t.things
}

// String returns a string representation of the structure
func (t cons) String() string {
	if len(t.Conditions()) == 0 {
		return ""
	}
	sb := strings.Builder{}
	for i, th := range t.things {
		if i > 0 {
			sb.WriteString(fmt.Sprintf(" %s ", th.Link))
		}
		sb.WriteString(fmt.Sprintf("%s %s[%t] %s (%s)", th.Key, th.Operator, th.Negate, th.Expression, th.ExprType.String()))
	}
	return sb.String()
}

// Add adds a thing to the collection
func (t *cons) Add(thing *Condition) {
	if t.Conditions() == nil {
		t.things = make([]*Condition, 0)
	}
	t.things = append(t.things, thing)
}

// Equals returns true if both instances are the same
func (t cons) Equals(t2 Conditions) bool {
	if t2 == nil {
		return false
	}
	if len(t.Conditions()) != len(t2.Conditions()) {
		return false
	}
	if len(t.Conditions()) == 0 {
		return true
	}

	return t.String() == t2.String()
}

// CheckMap checks the values in the map if they met the conditions
// for the corresponding key. Keys are checked case sensitive.
func (t *cons) CheckMap(m map[string]interface{}) (bool, error) {
	r := false
	for _, thing := range t.Conditions() {
		v, ok := m[thing.Key]
		if !ok {
			return false, fmt.Errorf("key not found: %s", thing.Key)
		}
		if err := t.check(thing, v, &r); err != nil {
			return false, err
		}
	}
	return r, nil
}

// CheckMapI checks the values in the ma if they met the conditions
// for the corresponding key. Keys are checked case insensitive.
func (t *cons) CheckMapI(m map[string]interface{}) (bool, error) {
	r := false
	for _, thing := range t.Conditions() {

		var v interface{}
		var found = false
		for k, val := range m {
			if strings.ToLower(thing.Key) == strings.ToLower(k) {
				v = val
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Errorf("key not found: %s", thing.Key)
		}

		if err := t.check(thing, v, &r); err != nil {
			return false, err
		}
	}
	return r, nil
}

// CheckStruct checks the fields of the struct if they
// met the conditions. Fields are case sensitive
func (t *cons) CheckStruct(s interface{}) (bool, error) {
	r := false

	st := reflect.ValueOf(s).Elem()
	for _, thing := range t.Conditions() {
		fv := st.FieldByName(thing.Key)
		if !fv.IsValid() {
			return false, fmt.Errorf("key not found: %s", thing.Key)
		}

		if err := t.check(thing, fv.Interface(), &r); err != nil {
			return false, err
		}
	}
	return r, nil
}

func (t *cons) AddDateFormat(format string) {
	t.dateFormats = append(t.dateFormats, format)
}

func (t *cons) check(thing *Condition, v interface{}, r *bool) error {

	x, err := t.checkVal(thing, v)
	if err != nil {
		return err
	}
	if thing.Link == LNK_AND {
		if thing.Negate {
			*r = *r && !x
		} else {
			*r = *r && x
		}
	} else if thing.Link == LNK_OR {
		if thing.Negate {
			*r = *r || !x
		} else {
			*r = *r || x
		}
	} else {
		*r = x
	}
	return nil
}

func (t *cons) checkVal(th *Condition, v interface{}) (bool, error) {
	switch v.(type) {
	case string:
		return checkString(th, v.(string))
	case int:
		return checkInt64(th, int64(v.(int)))
	case int8:
		return checkInt64(th, int64(v.(int8)))
	case int16:
		return checkInt64(th, int64(v.(int16)))
	case int32:
		return checkInt64(th, int64(v.(int32)))
	case int64:
		return checkInt64(th, v.(int64))
	case uint:
		return checkUint64(th, uint64(v.(int)))
	case uint8:
		return checkUint64(th, uint64(v.(uint8)))
	case uint16:
		return checkUint64(th, uint64(v.(uint16)))
	case uint32:
		return checkUint64(th, uint64(v.(uint32)))
	case uint64:
		return checkUint64(th, v.(uint64))
	case float32, float64:
		return checkFloat(th, v.(float64))
	case bool:
		return checkBool(th, v.(bool))
	case time.Time:
		return checkTime(th, v.(time.Time), t.dateFormats)
	}
	return false, fmt.Errorf("unsupported type: %T", v)
}

func checkString(t *Condition, s string) (bool, error) {
	switch t.Operator {
	case OP_EQ:
		return t.Expression == s, nil
	case OP_NEQ:
		return t.Expression != s, nil
	case OP_EQI:
		return strings.ToLower(t.Expression) == strings.ToLower(s), nil
	case OP_NEQI:
		return strings.ToLower(t.Expression) != strings.ToLower(s), nil
	case OP_GT:
		return t.Expression > s, nil
	case OP_GE:
		return t.Expression >= s, nil
	case OP_LT:
		return t.Expression < s, nil
	case OP_LE:
		return t.Expression <= s, nil
	case OP_RX, OP_RXN:
		rx, err := regexp.Compile(t.Expression)
		if err != nil {
			return false, err
		}
		m := rx.MatchString(s)
		if t.Operator == OP_RX {
			return m, nil
		}
		return !m, nil
	}
	return false, fmt.Errorf("opearator unsupported for string check: %s", t.Operator)
}

func checkTime(t *Condition, t2 time.Time, formats []string) (bool, error) {
	if t.ExprType != TIME {
		return checkString(t, fmt.Sprintf("%s", t2))
	}

	var tt time.Time
	var err error
	for _, f := range formats {
		tt, err = time.Parse(f, t.Expression)
		if err == nil {
			break
		}
	}
	if err != nil {
		return false, err
	}

	switch t.Operator {
	case OP_EQ, OP_EQI:
		return t2.Equal(tt), nil
	case OP_NEQ, OP_NEQI:
		return !t2.Equal(tt), nil
	case OP_GT:
		return t2.After(tt), nil
	case OP_GE:
		return t2.After(tt) || t2.Equal(tt), nil
	case OP_LT:
		return t2.Before(tt), nil
	case OP_LE:
		return t2.Before(tt) || t2.Equal(tt), nil
	}
	return false, fmt.Errorf("opearator unsupported for int check: %s", t.Operator)

}

func checkBool(t *Condition, b bool) (bool, error) {
	if t.ExprType != BOOLEAN {
		return checkString(t, fmt.Sprintf("%t", b))
	}
	j, err := strconv.ParseBool(t.Expression)
	if err != nil {
		return false, err
	}
	switch t.Operator {
	case OP_EQ, OP_EQI:
		return b == j, nil
	case OP_NEQ, OP_NEQI:
		return b != j, nil
	}
	return false, fmt.Errorf("opearator unsupported for bool check: %s", t.Operator)
}

func checkInt64(t *Condition, i int64) (bool, error) {
	if t.ExprType != INTEGER {
		return checkString(t, fmt.Sprintf("%d", i))
	}
	j, err := strconv.ParseInt(t.Expression, 10, 64)
	if err != nil {
		return false, err
	}
	switch t.Operator {
	case OP_EQ, OP_EQI:
		return i == j, nil
	case OP_NEQ, OP_NEQI:
		return i != j, nil
	case OP_GT:
		return i > j, nil
	case OP_GE:
		return i >= j, nil
	case OP_LT:
		return i < j, nil
	case OP_LE:
		return i <= j, nil
	}
	return false, fmt.Errorf("opearator unsupported for int check: %s", t.Operator)
}

func checkUint64(t *Condition, i uint64) (bool, error) {
	if t.ExprType != INTEGER {
		return checkString(t, fmt.Sprintf("%d", i))
	}
	j, err := strconv.ParseUint(t.Expression, 10, 64)
	if err != nil {
		return false, err
	}
	switch t.Operator {
	case OP_EQ, OP_EQI:
		return i == j, nil
	case OP_NEQ, OP_NEQI:
		return i != j, nil
	case OP_GT:
		return i > j, nil
	case OP_GE:
		return i >= j, nil
	case OP_LT:
		return i < j, nil
	case OP_LE:
		return i <= j, nil
	}
	return false, fmt.Errorf("opearator unsupported for uint check: %s", t.Operator)
}

func checkFloat(t *Condition, f float64) (bool, error) {
	if t.ExprType != FLOAT {
		return checkString(t, fmt.Sprintf("%f", f))
	}
	j, err := strconv.ParseFloat(t.Expression, 64)
	if err != nil {
		return false, err
	}
	switch t.Operator {
	case OP_EQ, OP_EQI:
		return f == j, nil
	case OP_NEQ, OP_NEQI:
		return f != j, nil
	case OP_GT:
		return f > j, nil
	case OP_GE:
		return f >= j, nil
	case OP_LT:
		return f < j, nil
	case OP_LE:
		return f <= j, nil
	}
	return false, fmt.Errorf("opearator unsupported for float check: %s", t.Operator)
}

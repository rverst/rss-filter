package goql

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Things interface {
	String() string
	Add(thing *Thing)
	Equals(thing Things) bool
	AddDateFormat(format string)
	CheckMap(m map[string]interface{}) (bool, error)
	CheckStruct(s interface{}) (bool, error)
	Things() []*Thing
}

type Thing struct {
	Link       Token
	Negate     bool
	Key        string
	Expression string
	ExprType   Token
	Operator   Token
}

type things struct {
	things      []*Thing
	dateFormats []string
}

func NewThings() *things {
	return &things{
		dateFormats: []string { time.RFC3339 },
	}
}

func (t things) Things() []*Thing {
	return t.things
}

// String returns a string representation of the structure
func (t things) String() string {
	if len(t.Things()) == 0 {
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
func (t *things) Add(thing *Thing) {
	if t.Things() == nil {
		t.things = make([]*Thing, 0)
	}
	t.things = append(t.things, thing)
}


// Equals returns true if both instances are the same
func (t things) Equals(t2 Things) bool {
	if t2 == nil {
		return false
	}
	if len(t.Things()) != len(t2.Things()) {
		return false
	}
	if len(t.Things()) == 0 {
		return true
	}

	return t.String() == t2.String()
}

func (t *things) CheckMap(m map[string]interface{}) (bool, error) {
	r := false
	for _, thing := range t.Things() {
		v, ok := m[thing.Key]
		if !ok {
			return false, fmt.Errorf("key not found: %s", thing.Key)
		}
		x, err := t.checkVal(thing, v)
		if err != nil {
			return x, err
		}
		if thing.Link == LNK_AND {
			if thing.Negate {
				r = r && !x
			} else {
				r = r && x
			}
		} else if thing.Link == LNK_OR {
			if thing.Negate {
				r = r || !x
			} else {
				r = r || x
			}
		} else {
			r = x
		}
	}
	return r, nil
}

func (t *things) CheckStruct(s interface{}) (bool, error)  {
	r := false

	st := reflect.ValueOf(s).Elem()

	for _, thing := range t.Things() {
		fv := st.FieldByName(thing.Key)
		if !fv.IsValid() {
			return false, fmt.Errorf("key not found: %s", thing.Key)
		}
		x, err := t.checkVal(thing, fv.Interface())
		if err != nil {
			return x, err
		}
		if thing.Link == LNK_AND {
			if thing.Negate {
				r = r && !x
			} else {
				r = r && x
			}
		} else if thing.Link == LNK_OR {
			if thing.Negate {
				r = r || !x
			} else {
				r = r || x
			}
		} else {
			r = x
		}
	}

	return r, nil
}


func (t *things) AddDateFormat(format string) {
	t.dateFormats = append(t.dateFormats, format)
}

func (t *things)checkVal(th *Thing, v interface{}) (bool, error) {
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

func checkString(t *Thing, s string) (bool, error) {
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

func checkTime(t *Thing, t2 time.Time, formats []string) (bool, error) {
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

func checkBool(t *Thing, b bool) (bool, error) {
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

func checkInt64(t *Thing, i int64) (bool, error) {
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

func checkUint64(t *Thing, i uint64) (bool, error) {
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

func checkFloat(t *Thing, f float64) (bool, error) {
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

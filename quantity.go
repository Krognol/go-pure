package pure

import (
	"regexp"
)

type Quantity struct {
	value string
}

func NewQuantity(val string) *Quantity {
	q := &Quantity{val}
	return q
}

func (q *Quantity) Unit() string {
	reg := regexp.MustCompile("([a-zA-Z_-]+[@#%/^.0-9]*)+")
	return reg.FindString(q.value)
}

func (q *Quantity) Value() string {
	reg := regexp.MustCompile("[0-9.,:;'-_]+")
	return reg.FindString(q.value)
}

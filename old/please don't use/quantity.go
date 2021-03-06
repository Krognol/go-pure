package pure

import (
	"regexp"
)

type Quantity struct {
	Value_ string
}

func NewQuantity(val string) *Quantity {
	q := &Quantity{val}
	return q
}

func (q *Quantity) Unit() string {
	reg := regexp.MustCompile("([a-zA-Z_-]+[@#%/^.0-9]*)+")
	return reg.FindString(q.Value_)
}

func (q *Quantity) Value() string {
	reg := regexp.MustCompile(`(\d+(\.\d+)?)`)
	return reg.FindString(q.Value_)
}

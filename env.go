package pure

import "os"

type Env struct {
	value string
}

func NewEnv(val string) *Env {
	return &Env{val}
}

func (e *Env) Expand() string {
	return os.ExpandEnv(e.value)
}

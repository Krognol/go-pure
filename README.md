# go-pure

The Pure specifications can be found [here](https://github.com/pureconfig/pureconfig).

# Usage
Pure file:
```
intproperty = 43

agroup.double = 1.23

agroup
    groupstring = "Hello, world!"

```

```go
package main

import (
	"io/ioutil"
	"os"

	"github.com/Krognol/pure"
)

type T struct {
	Property int `pure:"intproperty"`
	Group    *G  `pure:"agroup"`
}

type G struct {
	String string  `pure:"groupstring"`
	Double float64 `pure:"double"`
}

func main() {
	t := &T{0, &G{}}
	b, _ := ioutil.ReadFile("some-pure-file.pure")
	err := pure.Unmarshal(b, t)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	println(t.Property)     // => 42
	println(t.Group.String) // => "Hello, world!"
	println(t.Group.Double) // => 1.23
	os.Exit(0)
}
```

# Progress
- [x] Dot notation groups
- [x] Newline-tab groups
- [x] Regular properties
- [ ] Referencing
- [ ] Quantities
- [ ] Paths
- [ ] Environment variables
- [ ] Group Nesting? (Haven't tested)
- [ ] Arrays
- [ ] Schema support
- [ ] Include files
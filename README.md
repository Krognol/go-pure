# go-pure

The Pure specifications can be found [here](https://github.com/pureconfig/pureconfig).

# Usage
Pure file:
```
intproperty = 43

agroup.double = 1.23

agroup
    groupstring = "Hello, world!"

refstring => agroup.groupstring
refint => intproperty
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
	RefString string `pure:"refstring"`
	PropRef int `pure:"refint"`
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
	println(t.RefString)    // => "Hello, world!"
	println(t.PropRef)      // => 42
	os.Exit(0)
}
```
## Nesting

Pure file:
```
nested
	anotherone
		prop = "Hello, world!"
```

```go
package main

import (
	"github.com/Krognol/go-pure"
	"os"
	"io/ioutil"
)

type AnotherOne struct {
	String string `pure:"prop"`
}

type Nested struct {
	AnotherNested *AnotherOne `pure:"anotherone"`
}

type Base struct {
	Nested *Nested `pure:"nested"`
}

func main() {
	base := &Base{
		Nested: &Nested{
			AnotherNested: &AnotherOne{},
		},
	}

	b, _ := ioutil.ReadFile("nested-group-file.pure")

	err := pure.Unmarshal(b, base)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	println(base.Nested.AnotherNested.String) // => "Hello, world!"
	os.Exit(0)
}
```


# Progress
- [x] Dot notation groups
- [x] Newline-tab groups
- [x] Regular properties
- [x] Referencing
- [ ] Quantities
- [ ] Paths
- [ ] Environment variables
- [x] Group Nesting
- [ ] Arrays
- [ ] Schema support
- [ ] Include files
- [ ] Encoding to Pure format

# Contributing
1. Fork it ( https://github.com/Krognol/go-pure/fork )
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Add some feature')
4. Push to the branch (git push origin my-new-feature)
5. Create a new Pull Request
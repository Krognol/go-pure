# go-pure

The Pure specifications can be found [here](https://github.com/pureconfig/pureconfig).

# Setup

`go get -u github.com/Krognol/go-pure`

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

## Including files

Pure file to be included:
```
someproperty = 123
```

Main pure file:
```
%include ./someincludefile.pure

aProperty = "some \
			 weird text \
			 here or something"
```

Go program:

```go
package main

import (
	"os"
	"io/ioutil"
	"github.com/Krognol/pure"
)

type Include struct {
	// Property to be included
	SomeProperty int `pure:"someproperty"`

	// Base file Property
	AProperty string `pure:"aProperty"`
}


func main() {
	it := &Include{}
	b, _ := ioutil.ReadFile("./some-pure-file.pure")
	err := pure.Unmarshal(b, it)
	if err != nil {
		panic(err)
	}
	println(it.SomeProperty) // => 123
	println(it.AProperty)    // => "some weird text here or something"
}
```

## Quantities

Pure file:
```
quantity = 5m^2
```

Go program:
```go
package main

import (
	"os"
	"io/ioutil"
	"github.com/Krognol/go-pure"
)

type Q struct {
	Quantity *pure.Quantity `pure:"quantity"`
}

func main() {
	q := &Q{}
	b, _ := ioutil.ReadFile("./quantity.pure")
	err := pure.Unmarshal(b, q)
	if err != nil {
		panic(err)
	}
	println(q.Quantity.Value()) // => 5
	println(q.Quantity.Unit())  // => 'm^2'
}
```

## Environment variables

Pure file:

```
env = ${GOPATH}
```

Go program:

```go
package main

import (
	"os"
	"io/ioutil"
	"github.com/Krognol/go-pure"
)

type Env struct {
	E *pure.Env `pure:"env"`
}

func main() {
	e := &Env{}
	b, _ := ioutil.ReadFile("envfile.pure")
	err := pure.Unmarshal(b, e)
	if err != nil {
		penic(err)
	}
	println(e.E.Expand()) // => X:\your\go\path
	os.Exit(0)
}

```

## Paths

Pure file:
```
dir = ./some/directory/
file = ./some/directory/some/file.txt
```

Go program:
```go
package main

import(
	"io/ioutil"
	"os"
	"github.com/Krognol/go-pure"
)

type Dirs struct {
	Dir *pure.Path `pure:"dir"`
	File *pure.Path `pure:"file"`
}

func main() {
	dir := &Dirs{}
	b, _ := ioutil.ReadFile("./purefile.pure")
	err := pure.Unmarshal(b, dir)
	if err != nil {
		panic(err)
	}

	println(dir.Dir.Base()) // => 'directory'
	println(dir.File.FileExtension()) // => '.txt'
	os.Exit(0)
}
```

## Arrays

For now arrays only work for basic types (string, int, path...), and not for Groups.

Pure file:
```
array = [
	"Hello"
	"World!"
]

map = [
	int = 123
	anotherint = 321
]
```

Go program:

```go
package main

import(
	"os"
	"io/ioutil"
	"github.com/Krognol/go-pure"
)

type Array struct {
	Arr []string `pure:"array"`
	Map map[string]int `pure:"array"`
}

func main() {
	arr := &Array{Map: make(map[string]int)} // Very important to initialize the map before unmarshaling
	b, _ := ioutil.ReadFile("array-pure-file.pure")

	err := pure.Unmarshal(b, arr)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	println(arr.Arr[0])        // => "Hello"
	println(arr.Arr[1])        // => "World!"
	println(arr.Map["int"])    // => 123
	println(arr.Map["anotherint"]) // => 321
	os.Exit(0)
}

```


# Progress
- [x] Dot notation groups
- [x] Newline-tab groups
- [x] Regular properties
- [x] Referencing
- [x] Quantities
- [x] Paths
- [x] Environment variables
- [x] Group Nesting
- [ ] Arrays
- [ ] Schema support
- [x] Include files
- [ ] Encoding to Pure format
- [ ] Unquoted strings
- [x] Character escaping
- [x] Multiline values

# Contributing
1. Fork it ( https://github.com/Krognol/go-pure/fork )
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Add some feature')
4. Push to the branch (git push origin my-new-feature)
5. Create a new Pull Request
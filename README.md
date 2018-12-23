# injectgo

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://travis-ci.org/RivenZoo/injectgo.svg?branch=master)](https://travis-ci.org/RivenZoo/injectgo)
[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/RivenZoo/injectgo)

A golang inject module.

# Usage

```go
type Client struct {
    Address string
}

type Logger interface {
    Println(interface{})
}

type Model struct {
    Cli *Client `inject:""`
    Log Logger  `inject:""`
}

type dummyLogger struct{}

func (l dummyLogger) Println(s interface{}) {
    fmt.Println(s)
}

// ...

c := injectgo.NewContainer()

m := &Model{}

c.Provide(m, &Client{Address: "127.0.0.1:8080"})
c.ProvideFunc(injectgo.InjectFunc{
    Fn: func() (Logger, error) {
        return &dummyLogger{}, nil
    },
    Label: "",
})
c.Populate(nil)

m.Log.Println(m.Cli)
```

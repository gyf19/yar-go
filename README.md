# Yar RPC framework for Golang

A Go client for [Yar RPC framework](https://github.com/laruence/yar).

## Introduction

Yar is a light RPC framework for PHP written by Laruence. 
If you are looking for a Go for Yar framework, this project may
solve your problems.

## Usage

Use this client is very simple, just few codes:

```golang
var client, err = yar.Dial("tcp", Addr, "msgpack")
var reply = &Args{}
err := client.Call("Arith.Multiply", &Args{4, 5, "GO"}, reply)
fmt.Println(reply, err)
client.Close()
```

If you any questions, use [Issues](https://github.com/gyf19/yar-go/issues).

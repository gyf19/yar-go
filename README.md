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
Use this server is very simple, just few codes:

```golang
type Arith int

type Args struct {
	A, B int
	C    string
}

func (t *Arith) Multiply(args *Args, reply *Args) error {
	reply.A = args.A * args.B
	reply.C = args.C + "_hello"
	return nil
}

var server = yar.NewServer()
server.Register(new(Arith))
listener, err := net.Listen("tcp", ":12345")
if err != nil {
	return
}
server.Accept(listener)
```

Use this php client is very simple.

```php
$client = new Yar_Client('tcp://127.0.0.1:12345');
$arguments =  ['A'=>4,'B'=>5,'C'=>'php'];
$data = $client->__call("Arith.Multiply",$arguments);
var_dump($data);
```


If you any questions, use [Issues](https://github.com/gyf19/yar-go/issues).

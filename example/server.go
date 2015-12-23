package main

import (
	"github.com/gyf19/yar-go/yar"
	"flag"
	"fmt"
	"net"
	"runtime"
	"sync"
)


type Arith int

type Args struct {
	A, B int
	C    string
}

func init() {
}

func (t *Arith) Multiply(args *Args, reply *Args) error {
	reply.A = args.A * args.B
	reply.C = args.C + "_hello"
	return nil
}

var worker = runtime.NumCPU()

func main() {
	runtime.GOMAXPROCS(worker)

	var server = yar.NewServer()
	server.Register(new(Arith))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		listener, err := net.Listen("tcp", ":12345")
		if err != nil {
			fmt.Println(err)
			return
		}
		wg.Done()
		server.Accept(listener)
	}()
	wg.Wait()
	fmt.Scanln()

}

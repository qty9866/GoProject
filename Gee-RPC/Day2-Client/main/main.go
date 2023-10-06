package main

import (
	"fmt"
	geerpc "github.com/Hud9866/GeeRPC/Day2-Client"
	"log"
	"net"
	"sync"
	"time"
)

func StartServer(addr chan string) {
	// pick a free port
	listen, err := net.Listen("tcp", "localhost:9091")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on:", listen.Addr())
	addr <- listen.Addr().String()
	geerpc.Accept(listen)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go StartServer(addr)
	client, _ := geerpc.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}

package main

import (
	"encoding/json"
	"fmt"
	geerpc "github.com/Hud9866/GeeRPC/Day1-codec"
	"github.com/Hud9866/GeeRPC/Day1-codec/codec"
	"log"
	"net"
	"time"
)

func StartServer(addr chan string) {
	// pick a free port
	listen, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on:", listen.Addr())
	addr <- listen.Addr().String()
	geerpc.Accept(listen)
}

func main() {
	addr := make(chan string)
	go StartServer(addr)

	//in fact,the following code is like a simple gee-rpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	// send Options
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc := codec.NewJsonCodec(conn)
	// send request & receive response
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("geerpc req %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}

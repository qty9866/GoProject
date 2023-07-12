package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"
)

// 我们先看看标准库net/http如何处理一个请求。

func TestHttp(t *testing.T) {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/hello", helloHandler)
	//第二个参数则代表处理所有的HTTP请求的实例，nil 代表使用标准库中的实例处理。
	//第二个参数，则是我们基于net/http标准库实现Web框架的入口。
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
}

func helloHandler(w http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
	}
}

/*
我们设置了2个路由，/和/hello，分别绑定 indexHandler 和 helloHandler ，
根据不同的HTTP请求会调用不同的处理函数。访问/，响应是URL.Path = /，
而/hello的响应则是请求头(header)中的键值对信息。
*/

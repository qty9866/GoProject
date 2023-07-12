## Day2 Context
- 将路由 `router` 独立出来，方便之后增强。
- 设计上下文 `Context` ，封装 `Request` 和 `Response` ，提供对 `JSON`、`HTML` 等返回类型的支持

**使用实例展示**
```go
package main

func main() {
	r := gee.New()
	r.GET("/", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello gee</h1>")
	})
	r.GET("/hello", func(c *gee.Context) {
		// expect /hello?name=geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.POST("/login", func(c *gee.Context) {
		c.JSON(http.StatusOK, gee.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	r.Run(":8080")
}
```
- `Handler` 的参数变成了 `gee.Context` ，提供了查询 `Query/PostForm` 参数的功能。
- `gee.Context`封装了 `HTML` / `String` / `JSON` 函数，能够快速构造 `HTTP` 响应。


### 设计Context
#### 必要性
1. 对Web服务来说，无非是根据请求 `*http.Request` ，构造响应`http.ResponseWriter`。但是这两个对象提供的接口粒度太细，比如我们要构造一个完整的响应，需要考虑消息头(`Header`)和消息体(`Body`)，而 `Header` 包含了状态码(`StatusCode`)，消息类型(`ContentType`)等几乎每次请求都需要设置的信息。因此，如果不进行有效的封装，那么框架的用户将需要写大量重复，繁杂的代码，而且容易出错。针对常用场景，能够高效地构造出 `HTTP` 响应是一个好的框架必须考虑的点

用返回JSON做比较 感受一下

**封装前：**
```go
obj := map[string]interface{}{
    "name": "Hud9866",
    "password": "1234",
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
encoder := json.NewEncoder(w)
if err := encoder.Encode(obj); err != nil {
    http.Error(w, err.Error(), 500)
}
```

**封装后:**
	
```go
c.JSON(http.StatusOK, gee.H{
    "username": c.PostForm("username"),
    "password": c.PostForm("password"),
})
```
2. 针对使用场景，封装`*http.Request`和`http.ResponseWriter`的方法，简化相关接口的调用，只是设计 `Context` 的原因之一。对于框架来说，还需要支撑额外的功能。例如，将来解析动态路由`/hello/:name`，参数`:name`的值放在哪呢？再比如，框架需要支持中间件，那中间件产生的信息放在哪呢？`Context` 随着每一个请求的出现而产生，请求的结束而销毁，和当前请求强相关的信息都应由 `Context` 承载。因此，设计 `Context` 结构，扩展性和复杂性留在了内部，而对外简化了接口。路由的处理函数，以及将要实现的中间件，参数都统一使用 `Context` 实例， `Context` 就像一次会话的百宝箱，可以找到任何东西。
    
    - 代码最开头，给`map[string]interface{}`起了一个别名`gee.H`，构建`JSON`数据时，显得更简洁。
    - `Context`目前只包含了`http.ResponseWriter`和`*http.Request`，另外提供了对 Method 和 Path 这两个常用属性的直接访问。
    - 提供了访问`Query`和`PostForm`参数的方法。
    - 提供了快速构造`String`/`Data`/`JSON`/`HTML`响应的方法

### 路由(Router)

我们将和路由相关的方法和结构提取了出来，放到了一个新的文件中`router.go`，方便我们下一次对 `router` 的功能进行增强，例如提供动态路由的支持。 `router` 的 `handle` 方法作了一个细微的调整，即 `handler` 的参数，变成了 `Context`。


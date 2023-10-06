## day1 服务端与消息编码
```go
type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	Error         string
}
```
- ServiceMethod 是服务名和方法名，通常与 Go 语言中的结构体和方法相映射。
- Seq 是请求的序号，也可以认为是某个请求的 ID，用来区分不同的请求。
- Error 是错误信息，客户端置为空，服务端如果如果发生错误，将错误信息置于 Error 中。

**消息体进行编解码的接口 Codec，抽象出接口是为了实现不同的 Codec 实例**
```go
type Codec interface {
    io.Closer
    ReadHeader(*Header) error
    ReadBody(interface{}) error
    Write(*Header, interface{}) error
}
```
### 通信过程
对于 GeeRPC 来说，目前需要协商的唯一一项内容是消息的编解码方式。我们将这部分信息，放到结构体 Option 中承载。目前，已经进入到服务端的实现阶段了。

GeeRPC 客户端固定采用 JSON 编码 Option，后续的 header 和 body 的编码方式由 Option 中的 CodeType 指定，服务端首先使用 JSON 解码 Option，然后通过 Option 的 CodeType 解码剩余的内容。即报文将以这样的形式发送：

### 服务端的实现
- 首先定义了结构体 Server，没有任何的成员字段。
  ```go
  // Server represents an RPC Server.
	 type Server struct{}
  ```
- 实现了 Accept 方式，net.Listener 作为参数，for 循环等待 socket 连接建立，并开启子协程处理，处理过程交给了 ServerConn 方法。
    ```go
    func (server *Server) Accept(lis net.Listener) {
        for {
            conn, err := lis.Accept()
            if err != nil {
                log.Println("rpc server: accept error:", err)
                return
            }
            go server.ServeConn(conn)
        }
    }
    ```
- `DefaultServer` 是一个默认的 Server 实例，主要为了用户使用方便。


`ServeConn` 的实现就和之前讨论的通信过程紧密相关了，首先使用 `json.NewDecoder` 反序列化得到 `Option` 实例，检查 `MagicNumber` 和 `CodeType` 的值是否正确。然后根据 `CodeType` 得到对应的消息编解码器，接下来的处理交给 `serverCodec`。

`serveCodec` 的过程非常简单。主要包含三个阶段

- 读取请求 `readRequest`
- 处理请求 `handleRequest`
- 回复请求 `sendResponse`

  
之前提到过，在一次连接中，允许接收多个请求，即多个 request header 和 request body，因此这里使用了 for 无限制地等待请求的到来，直到发生错误（例如连接被关闭，接收到的报文有问题等），这里需要注意的点有三个：

- handleRequest 使用了协程并发执行请求。
- 处理请求是并发的，但是回复请求的报文必须是逐个发送的，并发容易导致多个回复报文交织在一起，客户端无法解析。在这里使用锁(sending)保证。
- 尽力而为，只有在 header 解析失败时，才终止循环。

## Call 的设计
对 `net/rpc` 而言，一个函数需要能够被远程调用，需要满足如下五个条件：
```
the method’s type is exported.
the method is exported.
the method has two arguments, both exported (or builtin) types.
the method’s second argument is a pointer.
the method has return type error.
```
**更直观一些：**

`func (t *T) MethodName(argType T1, replyType *T2) error
`
根据上述要求，首先我们封装了结构体 Call 来承载一次 RPC 调用所需要的信息.

day2-client/client.go
```go
// Call represents an active RPC.
type Call struct {
    Seq           uint64
    ServiceMethod string      // format "<service>.<method>"
    Args          interface{} // arguments to the function
    Reply         interface{} // reply from the function
    Error         error       // if error occurs, it will be set
    Done          chan *Call  // Strobes when call is complete.
}

func (call *Call) done() {
    call.Done <- call
}
```
为了支持异步调用，Call 结构体中添加了一个字段 Done，Done 的类型是 chan *Call，当调用结束时，会调用 call.done() 通知调用方。

Client 的字段比较复杂：

- cc 是消息的编解码器，和服务端类似，用来序列化将要发送出去的请求，以及反序列化接收到的响应。
- sending 是一个互斥锁，和服务端类似，为了保证请求的有序发送，即防止出现多个请求报文混淆。
- header 是每个请求的消息头，header 只有在请求发送时才需要，而请求发送是互斥的，因此每个客户端只需要一个，声明在 Client 结构体中可以复用。
- seq 用于给发送的请求编号，每个请求拥有唯一编号。
- pending 存储未处理完的请求，键是编号，值是 Call 实例。
- closing 和 shutdown 任意一个值置为 true，则表示 Client 处于不可用的状态，但有些许的差别，closing 是用户主动关闭的，即调用 Close 方法，而 shutdown 置为 true 一般是有错误发生
```go
type Client struct {
	cc       codec.Codec
	opt      *Option
	sending  sync.Mutex // protect following
	header   codec.Header
	mu       sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called close
	shutdown bool // server has told us to stop
}
```

紧接着，实现和 Call 相关的三个方法。

```go
func (client *Client) registerCall(call *Call) (uint64, error) {
    client.mu.Lock()
    defer client.mu.Unlock()
    if client.closing || client.shutdown {
    return 0, ErrShutdown
}
    call.Seq = client.seq
    client.pending[call.Seq] = call
    client.seq++
    return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
    client.mu.Lock()
    defer client.mu.Unlock()
    call := client.pending[seq]
    delete(client.pending, seq)
    return call
}

func (client *Client) terminateCalls(err error) {
    client.sending.Lock()
    defer client.sending.Unlock()
    client.mu.Lock()
    defer client.mu.Unlock()
    client.shutdown = true
    for _, call := range client.pending {
        call.Error = err
        call.done()
    }   
}
```
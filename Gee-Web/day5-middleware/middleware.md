## Day5 MiddleWare

- **设计并实现 Web 框架的中间件(Middlewares)机制。**
- **实现通用的Logger中间件，能够记录请求到响应所花费的时间**

### 中间件是什么

中间件(middlewares)，简单说，就是非业务的技术类组件。Web 框架本身不可能去理解所有的业务，因而不可能实现所有的功能。因此，框架需要有一个插口，允许用户自己定义功能，嵌入到框架中，仿佛这个功能是框架原生支持的一样。因此，对中间件而言，需要考虑2个比较关键的点：

插入点在哪？使用框架的人并不关心底层逻辑的具体实现，如果插入点太底层，中间件逻辑就会非常复杂。如果插入点离用户太近，那和用户直接定义一组函数，每次在 Handler 中手工调用没有多大的优势了。
中间件的输入是什么？中间件的输入，决定了扩展能力。暴露的参数太少，用户发挥空间有限。
那对于一个 Web 框架而言，中间件应该设计成什么样呢？接下来的实现，基本参考了 Gin 框架。

### 中间件设计

Gee 的中间件的定义与路由映射的 `Handler` 一致，处理的输入是 `Context` 对象。**插入点是框架接收到请求初始化`Context`对象后**，允许用户使用自己定义的中间件做一些额外的处理，例如记录日志等，以及对`Context`进行二次加工。另外通过调用`(*Context).Next()`函数，中间件可等待用户自己定义的 `Handler` 处理结束后，再做一些额外的操作，例如计算本次处理所用时间等。即 Gee 的中间件支持用户在请求被处理的前后，做一些额外的操作。举个例子，我们希望最终能够支持如下定义的中间件，c.Next()表示等待执行其他的中间件或用户的Handler：
```go
func Logger() HandlerFunc {
	return func(c *Context) {
		// Start timer
		t := time.Now()
		// Process request
		c.Next()
		// Calculate resolution time
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
```

另外，支持设置多个中间件，依次进行调用

我们上一篇文章分组控制 `Group Control` 中讲到，中间件是应用在 `RouterGroup` 上的，应用在最顶层的 `Group`，相当于作用于全局，所有的请求都会被中间件处理。那为什么不作用在每一条路由规则上呢？作用在某条路由规则，那还不如用户直接在 Handler 中调用直观。**只作用在某条路由规则的功能通用性太差，不适合定义为中间件。**

我们之前的框架设计是这样的，当接收到请求后，匹配路由，该请求的所有信息都保存在 `Context` 中。中间件也不例外，接收到请求后，应查找所有应作用于该路由的中间件，保存在 `Context` 中，依次进行调用。为什么依次调用后，还需要在Context中保存呢？**因为在设计中，中间件不仅作用在处理流程前，也可以作用在处理流程后，即在用户定义的 Handler 处理完毕后，还可以执行剩下的操作。**

为此，我们给Context添加了2个参数，定义了Next方法：

```go
type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
	// middleware
	handlers []HandlerFunc
	index    int
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Path:   req.URL.Path,
		Method: req.Method,
		Req:    req,
		Writer: w,
		index:  -1,
	}
}

func (c *Context) Next() {
    c.index++
    s := len(c.handlers)
    for ; c.index < s; c.index++ {
        c.handlers[c.index](c)
    }
}
```

`index` 是记录当前执行到第几个中间件，当在中间件中调用 `Next` 方法时，控制权交给了下一个中间件，直到调用到最后一个中间件，然后再从后往前，调用每个中间件在Next方法之后定义的部分。如果我们将用户在映射路由时定义的Handler添加到c.handlers列表中，结果会怎么样呢？想必你已经猜到了。
```go
func A(c *Context) {
    part1
    c.Next()
    part2
}
func B(c *Context) {
    part3
    c.Next()
    part4
}
```
假设我们应用了中间件 A 和 B，和路由映射的 Handler。c.handlers是这样的[A, B, Handler]，c.index初始化为-1。调用c.Next()，接下来的流程是这样的：

- c.index++，c.index 变为 0 
- 0 < 3，调用 c.handlers[0]，即 A 
- 执行 part1，调用 c.Next()
- c.index++，c.index 变为 1 
- 1 < 3，调用 c.handlers[1]，即 B
- 执行 part3，调用 c.Next()
- c.index++，c.index 变为 2 
- 2 < 3，调用 c.handlers[2]，即Handler 
- Handler 调用完毕，返回到 B 中的 part4，执行 part4 
- part4 执行完毕，返回到 A 中的 part2，执行 part2 
- part2 执行完毕，结束。

一句话说清楚重点，最终的顺序是`part1 -> part3 -> Handler -> part 4 -> part2`。恰恰满足了我们对中间件的要求，接下来看调用部分的代码，就能全部串起来了。

### 代码实现

**定义Use函数，将中间件应用到某个 Group 。**

```go
// Use is defined to add middleware to the group
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, req)
	c.handlers = middlewares
	engine.router.handle(c)
}
```

`ServeHTTP` 函数也有变化，当我们接收到一个具体请求时，要判断该请求适用于哪些中间件，在这里我们简单通过 URL 的前缀来判断。得到中间件列表后，赋值给 `c.handlers`

`handle` 函数中，将从路由匹配得到的 `Handler` 添加到 `c.handlers`列表中，执行`c.Next()`。

**gee/router.go**
```go
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)

	if n != nil {
		key := c.Method + "-" + n.pattern
		c.Params = params
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	c.Next()
}
```
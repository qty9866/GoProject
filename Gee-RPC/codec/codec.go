package codec

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"io"
)

// 消息编解码相关的代码

/*ServiceMethod 是服务名和方法名，通常与 Go 语言中的结构体和方法相映射。
Seq 是请求的序号，也可以认为是某个请求的 ID，用来区分不同的请求。
Error 是错误信息，客户端置为空，服务端如果如果发生错误，将错误信息置于 Error 中。*/

type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

// Codec 抽象出对消息体进行编码的接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodeFunc func(closer io.ReadWriteCloser) Codec
type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodeFuncMap map[Type]NewCodeFunc

func init() {
	NewCodeFuncMap = make(map[Type]NewCodeFunc)
	NewCodeFuncMap[GobType] = NewGobCodec
	NewCodeFuncMap[JsonType] = NewJsonCodec
}

// GobCodec buf 是为了防止阻塞而创建的带缓冲的 `Writer`，一般这么做能提升性能。
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

type JsonCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *json.Decoder
	enc  *json.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

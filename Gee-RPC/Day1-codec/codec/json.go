package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

// GobCodec buf 是为了防止阻塞而创建的带缓冲的 `Writer`，一般这么做能提升性能。
type JsonCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *json.Decoder
	enc  *json.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &JsonCodec{
		conn: conn,
		buf:  buf,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(buf),
	}
}

func (c *JsonCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *JsonCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *JsonCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc Codec:gob error encoding Header")
		return
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc Codec:gob error encoding Body")
		return
	}
	return nil
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}

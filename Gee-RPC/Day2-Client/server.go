package geerpc

import (
	"encoding/json"
	"fmt"
	"github.com/Hud9866/GeeRPC/Day2-Client/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this is a gee-rpc request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.JsonType,
}

// Server Represents an RPC Server
type Server struct{}

// NewServer returns a new *Server
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server
var DefaultServer = NewServer()

// Accept accepts connections on the listener and servers requests
// for each incoming connection
func (s *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go s.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

/*
ServeConn 的实现就和之前讨论的通信过程紧密相关了，首先使用 json.NewDecoder
反序列化得到 Option 实例，检查 MagicNumber 和 CodeType 的值是否正确。然后根据 CodeType
得到对应的消息编解码器，接下来的处理交给 serverCodec。
*/
func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server:options error", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server:invalid magic number %x:", opt.MagicNumber)
		return
	}
	f := codec.NewCodeFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server:invalid codec Type: %s", opt.CodecType)
		return
	}
	s.serveCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when errors occur
var invalidRequest = struct{}{}

func (s *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)
	for {
		req, err := s.readRequest(cc)
		if err != nil {
			if req == nil {
				break // it is not possible to recover,so close the connection
			}
			req.h.Error = err.Error()
			s.SendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go s.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

/*
serveCodec 的过程非常简单。主要包含三个阶段
 - 读取请求 readRequest
 - 处理请求 handleRequest
 - 回复请求 sendResponse
*/

// request store all information of a call
type request struct {
	h           *codec.Header
	argv, reply reflect.Value
}

func (s *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (s *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := s.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// todo: we don't know the type of request argv,now we just suppose it is string
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server : read argv err:", err)
	}
	return req, nil
}

func (s *Server) SendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error")
	}
}

func (s *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO, should call registered rpc methods to get the right replyv
	// day 1, just print argv and send a hello message
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	// 打印序列号
	req.reply = reflect.ValueOf(fmt.Sprintf("gee-rpc resp %d", req.h.Seq))
	s.SendResponse(cc, req.h, req.reply.Interface(), sending)
}

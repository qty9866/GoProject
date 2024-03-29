package base2

import (
	"fmt"
	"net/http"
	"testing"
)

// Engine is the uni handler for all requests
type Engine struct{}

func (engine *Engine) ServerHttp(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/":
		fmt.Fprintf(w, "URL.Path = %q\n", req.URL.Path)
	case "/hello":
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	default:
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}

func TestServerHttp(t *testing.T) {

}

package http

import (
	"fmt"
	"github.com/gorilla/mux"
	fts "github.com/sjqzhang/killgrave/fakehttpserver/server/http"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
)

type MockHttpServer struct {
	fakeHttpPort    int      `json:"fake_http_port"`
	httpProxyPort   int      `json:"http_proxy_port"`
	allowProxyHosts []string `json:"allow_proxy_hosts"`
	mockApiDir      string   `json:"mock_api_dir"`
	handler         http.Handler
	reqMap          map[string]Response
	lock            sync.Mutex
}

// Request represent the structure of real request
type Request struct {
	Host     string `json:"host"`
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
	//Params   *map[string]string `json:"params"`
	//Headers  *map[string]string `json:"headers"`
}

// Response represent the structure of real response
type Response struct {
	Status  int                `json:"status"`
	Body    string             `json:"body"`
	Headers *map[string]string `json:"headers"`
}

type httpHandler struct {
	allowProxyHosts []string
	mockHttpServer  *MockHttpServer
}

func NewMockHttpServer(mockJSONDir string, allowProxyHosts []string) *MockHttpServer {

	server := &MockHttpServer{
		fakeHttpPort:    23433,
		httpProxyPort:   23435,
		allowProxyHosts: allowProxyHosts,
		mockApiDir:      mockJSONDir,
		reqMap:          make(map[string]Response),
		lock:            sync.Mutex{},
	}
	hander := &httpHandler{
		allowProxyHosts: allowProxyHosts,
		mockHttpServer:  server,
	}
	server.handler = hander

	return server

}

func (m *httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	key := fmt.Sprintf("#%v_#%v_#%v", req.Host, strings.ToUpper(req.Method), req.URL.Path)
	if rsp, ok := m.mockHttpServer.reqMap[key]; ok {
		if rsp.Headers != nil {
			for k, v := range *rsp.Headers {
				resp.Header().Set(k, v)
			}
		}
		resp.Write([]byte(rsp.Body))
		resp.WriteHeader(rsp.Status)
		return
	}
	uri, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%v", m.mockHttpServer.fakeHttpPort))
	if err != nil {
		panic(err)
	}
	for _, host := range m.mockHttpServer.allowProxyHosts {
		if host == req.Host {
			httputil.NewSingleHostReverseProxy(uri).ServeHTTP(resp, req)
			return
		}
	} //not pass proxy
	client := http.Client{}
	client.Transport = &http.Transport{}
	req.RequestURI = ""
	rsp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	io.Copy(resp, rsp.Body)
	defer rsp.Body.Close()
}

func (m *MockHttpServer) DisableMockHttp() {
	os.Setenv("HTTP_PROXY", "")
	os.Setenv("HTTPS_PROXY", "")
}

func (m *MockHttpServer) SetCustomHttpHandler(handler http.Handler) {
	m.handler = handler
}

func (m *MockHttpServer) SetReqRspHandler(reqHander func(req *Request, rsp *Response)) {
	req, rsp := m.newReqToResponse()
	reqHander(&req, &rsp)
	m.setReqToResponse(req, rsp)
}

func (m *MockHttpServer) setReqToResponse(req Request, rsp Response) {
	m.lock.Lock()
	defer m.lock.Unlock()
	key := fmt.Sprintf("#%v_#%v_#%v", req.Host, strings.ToUpper(req.Method), req.Endpoint)
	m.reqMap[key] = rsp
}
func (m *MockHttpServer) newReqToResponse() (Request, Response) {
	var req Request
	var rsp Response
	req.Method = "GET"
	rsp.Status = 200
	header := make(map[string]string)
	rsp.Headers = &header
	return req, rsp
}

func (m *MockHttpServer) InitMockHttpServer() {
	os.Setenv("HTTP_PROXY", fmt.Sprintf("http://127.0.0.1:%v", m.httpProxyPort))
	os.Setenv("HTTPS_PROXY", fmt.Sprintf("http://127.0.0.1:%v", m.httpProxyPort))

	go func() {
		err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", m.httpProxyPort), m.handler)
		if err != nil {
			panic(err)
		}
	}()
	router := mux.NewRouter()
	httpServer := fts.NewServer(m.mockApiDir, router, &http.Server{Addr: fmt.Sprintf("0.0.0.0:%v", m.fakeHttpPort), Handler: router}, &fts.Proxy{}, false)
	err := httpServer.Build()
	if err != nil {
		panic(err)
	}
	httpServer.Run()

}

package mockhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type MockHttpServer struct {
	fakeHttpPort    int      `json:"fake_http_port"`
	httpProxyPort   int      `json:"http_proxy_port"`
	allowProxyHosts []string `json:"allow_proxy_hosts"`
	mockApiDir      string   `json:"mock_api_dir"`
	handler         http.Handler
	reqMap          map[string]Response
	lock            sync.Mutex
	mockReqRsp      []reqrsp
	closeFunc       func()
	once            sync.Once
}

type reqrsp struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

var consoleLog = log.New(os.Stdout, "[gmock.mockhttp] ", log.LstdFlags)

// Request represent the structure of real Request
type Request struct {
	Host     string `json:"host"`
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
	//Body     string `json:"body"`
	//Params   *map[string]string `json:"params"`
	//Headers  *map[string]string `json:"headers"`
}

// Response represent the structure of real Response
type Response struct {
	Status           int                `json:"status"`
	Body             string             `json:"body"`
	Headers          *map[string]string `json:"headers"`
	DelayMillisecond int                `json:"delayMillisecond"`
	Handler          func(resp http.ResponseWriter, req *http.Request)
}

type httpHandler struct {
	allowProxyHosts []string
	mockHttpServer  *MockHttpServer
}

type bufferReadWrite struct {
	buf  *bytes.Buffer
	head map[string][]string
}

func (b *bufferReadWrite) Header() http.Header {
	return b.head
}

func (b *bufferReadWrite) Write(bu []byte) (int, error) {

	return b.buf.Write(bu)
}
func (b *bufferReadWrite) Read() []byte {

	return b.buf.Bytes()
}

func (b *bufferReadWrite) WriteHeader(statusCode int) {

}

func newBuffer() *bufferReadWrite {
	return &bufferReadWrite{
		buf:  &bytes.Buffer{},
		head: make(map[string][]string, 10),
	}

}

func initReqRsp(mockJSONDir string) ([]reqrsp, error) {
	var rs []reqrsp
	fis, err := ioutil.ReadDir(mockJSONDir)
	if err == nil {
		for _, fi := range fis {
			data, err := ioutil.ReadFile(mockJSONDir + "/" + fi.Name())
			if err == nil {
				var reqs []reqrsp
				err = json.Unmarshal(data, &reqs)
				if err == nil {
					rs = append(rs, reqs...)
				}
			}
		}
	}
	return rs, nil
}

func NewMockHttpServer(httpServerPort int, mockJSONDir string, allowProxyHosts []string) *MockHttpServer {
	rs, _ := initReqRsp(mockJSONDir)
	server := &MockHttpServer{
		fakeHttpPort:    23433,
		httpProxyPort:   httpServerPort,
		allowProxyHosts: allowProxyHosts,
		mockApiDir:      mockJSONDir,
		reqMap:          make(map[string]Response),
		lock:            sync.Mutex{},
		mockReqRsp:      rs,
		closeFunc:       nil,
		once:            sync.Once{},
	}
	hander := &httpHandler{
		allowProxyHosts: allowProxyHosts,
		mockHttpServer:  server,
	}
	server.handler = hander

	return server

}

func (m *httpHandler) compareStrTrimBlank(str1 string, str2 string) bool {
	trimStr1 := strings.Trim(strings.TrimSpace(str1), "\n")
	trimStr2 := strings.Trim(strings.TrimSpace(str2), "\n")
	return trimStr1 == trimStr2
}

func (m *httpHandler) getRequest(rr *http.Request) reqrsp {
	var r reqrsp
	//reqBody := m.getRequestBody(rr)
	for _, req := range m.mockHttpServer.mockReqRsp {
		if req.Request.Endpoint == rr.URL.Path &&
			req.Request.Host == rr.Host &&
			strings.ToUpper(req.Request.Method) == strings.ToUpper(rr.Method) {
			return req
		}
	}
	for _, req := range m.mockHttpServer.mockReqRsp {
		exp, err := regexp.Compile(fmt.Sprintf("%v", req.Request.Endpoint))
		if err != nil {
			consoleLog.Println(err)
			continue
		}
		if exp.MatchString(rr.URL.Path) && req.Request.Host == rr.Host && strings.ToUpper(req.Request.Method) == strings.ToUpper(rr.Method) {
			return req
		}
	}
	return r
}

func (m *httpHandler) getRequestBody(req *http.Request) string {
	if req.Body == nil {
		return ""
	}
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return ""
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return string(b)
}

func (m *httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var body string
	bodyPrt := &body
	var respStatus int
	respStatusPtr := &respStatus
	buff := newBuffer()
	defer func() {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			consoleLog.Println(err)
		}
		consoleLog.Println(fmt.Sprintf("\033[32m <Request> Method:'%v'  RequestURI:%v Request Body:%v \u001B[0m", req.Method, req.RequestURI, string(data)))
		consoleLog.Println(fmt.Sprintf("\u001B[33m <Response> Method:'%v' Status:%v  RequestURI:%v Response Body:%v \u001B[0m", req.Method, *respStatusPtr, req.URL, *bodyPrt))
	}()
	//reqBody := m.getRequestBody(req)
	//key := fmt.Sprintf("#%v_#%v_#%v_#%v", req.Host, strings.ToUpper(req.Method), req.URL.Path)
	key := fmt.Sprintf("#%v_#%v_#%v", req.Host, strings.ToUpper(req.Method), req.URL.Path)
	if rsp, ok := m.mockHttpServer.reqMap[key]; ok {
		if rsp.Headers != nil {
			for k, v := range *rsp.Headers {
				resp.Header().Set(k, v)
			}
		}
		if rsp.Handler != nil {

			rsp.Handler(buff, req)
			resp.Write([]byte(buff.buf.String()))
			body = buff.buf.String()
			respStatus = rsp.Status
			if rsp.DelayMillisecond > 0 {
				time.Sleep(time.Microsecond * time.Duration(rsp.DelayMillisecond))
			}
			return
		}
		resp.WriteHeader(rsp.Status)
		resp.Write([]byte(rsp.Body))
		body = rsp.Body
		respStatus = rsp.Status
		if rsp.DelayMillisecond > 0 {
			time.Sleep(time.Microsecond * time.Duration(rsp.DelayMillisecond))
		}
		return
	}
	//uri, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%v", m.mockHttpServer.fakeHttpPort))
	//if err != nil {
	//	panic(err)
	//}
	f := false
	for _, host := range m.mockHttpServer.allowProxyHosts {
		if host == req.Host {
			f = true
			r := m.getRequest(req)
			if r.Request.Endpoint != "" {
				if r.Response.Handler != nil {
					r.Response.Handler(buff, req)
					resp.Write([]byte(buff.buf.String()))
					body = buff.buf.String()
					respStatus = r.Response.Status
					if r.Response.DelayMillisecond > 0 {
						time.Sleep(time.Microsecond * time.Duration(r.Response.DelayMillisecond))
					}
					return
				}
				resp.WriteHeader(r.Response.Status)
				resp.Write([]byte(r.Response.Body))
				body = r.Response.Body
				respStatus = r.Response.Status
				if r.Response.DelayMillisecond > 0 {
					time.Sleep(time.Microsecond * time.Duration(r.Response.DelayMillisecond))
				}
				return
			}
		}
	}
	if f {
		resp.WriteHeader(404)
		resp.Write([]byte("404 Not Found"))
		return
	}

	r := Request{
		Host:     req.Host,
		Method:   req.Method,
		Endpoint: req.URL.Path,
		//Body:     reqBody,
	}

	consoleLog.Println(fmt.Sprintf("\033[31m <ERROR> %v not match, Please check request config is correct ? \u001B[0m", r))

	//not pass proxy
	client := http.Client{}
	client.Transport = &http.Transport{}
	//req.RequestURI = ""
	rsp, err := client.Do(req)
	if err != nil {
		consoleLog.Println(err)
		return
	}
	io.Copy(resp, rsp.Body)
	defer rsp.Body.Close()
}

func (m *MockHttpServer) DisableMockHttp() {
	os.Setenv("HTTP_PROXY", "")
	os.Setenv("HTTPS_PROXY", "")
}

func (m *MockHttpServer) IsAlive() bool {
	_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%v", m.httpProxyPort))
	if err == nil {
		return true
	}
	return false
}

func (m *MockHttpServer) Stop() {
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

func (m *MockHttpServer) Start() (closeFunc func()) {
	var handler httpHandler
	handler.mockHttpServer = m
	handler.allowProxyHosts = m.allowProxyHosts
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.ServeHTTP)
	server := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", m.httpProxyPort), Handler: mux}
	go func() {
		// recover the http server
		defer func() {
			if err := recover(); err != nil {
				consoleLog.Println(err)
			}
		}()
		err := server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	i := 0
	for {
		time.Sleep(time.Second)
		if m.IsAlive() {
			break
		}
		i++
		if i > 10 {
			break
		}
	}
	m.closeFunc = func() {
		if err := server.Shutdown(context.Background()); err != nil {
			fmt.Printf("HTTP server Shutdown: %v", err)
		}
	}
	return m.closeFunc
}

//func (m *MockHttpServer) SetCustomHttpHandler(handler http.Handler) {
//	m.handler = handler
//}

func (m *MockHttpServer) SetReqRspHandler(reqHander func(req *Request, rsp *Response)) {
	req, rsp := m.newReqToResponse()
	reqHander(&req, &rsp)
	m.setReqToResponse(req, rsp)
}

func (m *MockHttpServer) setReqToResponse(req Request, rsp Response) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if req.Host== "" {
		req.Host = fmt.Sprintf("%v:%v", "127.0.0.1",m.httpProxyPort)
	}
	key := fmt.Sprintf("#%v_#%v_#%v", req.Host, strings.ToUpper(req.Method), req.Endpoint)
	//key := fmt.Sprintf("#%v_#%v_#%v", req.Host, strings.ToUpper(req.Method), req.Endpoint)
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

func (m *MockHttpServer) InitMockHttpServer() func() {
	os.Setenv("HTTP_PROXY", fmt.Sprintf("http://127.0.0.1:%v", m.httpProxyPort))
	os.Setenv("HTTPS_PROXY", fmt.Sprintf("http://127.0.0.1:%v", m.httpProxyPort))
	if m.IsAlive() {
		consoleLog.Println(fmt.Sprintf("server is running. listen on:%v", m.httpProxyPort))
		return m.closeFunc
	}
	return m.Start()
}

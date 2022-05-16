package mockhttp

import (
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
	once            sync.Once
}

type reqrsp struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

// Request represent the structure of real Request
type Request struct {
	Host     string `json:"host"`
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
	//Params   *map[string]string `json:"params"`
	//Headers  *map[string]string `json:"headers"`
}

// Response represent the structure of real Response
type Response struct {
	Status  int                `json:"status"`
	Body    string             `json:"body"`
	Headers *map[string]string `json:"headers"`
}

type httpHandler struct {
	allowProxyHosts []string
	mockHttpServer  *MockHttpServer
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

func NewMockHttpServer(mockJSONDir string, allowProxyHosts []string) *MockHttpServer {
	rs, _ := initReqRsp(mockJSONDir)
	server := &MockHttpServer{
		fakeHttpPort:    23433,
		httpProxyPort:   23435,
		allowProxyHosts: allowProxyHosts,
		mockApiDir:      mockJSONDir,
		reqMap:          make(map[string]Response),
		lock:            sync.Mutex{},
		mockReqRsp:      rs,
		once:            sync.Once{},
	}
	hander := &httpHandler{
		allowProxyHosts: allowProxyHosts,
		mockHttpServer:  server,
	}
	server.handler = hander

	return server

}

func (m *httpHandler) getRequest(rr *http.Request) reqrsp {
	var r reqrsp
	for _, req := range m.mockHttpServer.mockReqRsp {
		if req.Request.Endpoint == rr.URL.Path && req.Request.Host == rr.Host && strings.ToUpper(req.Request.Method) == strings.ToUpper(rr.Method) {
			return req
		}
	}
	for _, req := range m.mockHttpServer.mockReqRsp {
		exp, err := regexp.Compile(fmt.Sprintf("%v", req.Request.Endpoint))
		if err != nil {
			log.Println(err)
			continue
		}
		if exp.MatchString(rr.URL.Path) && req.Request.Host == rr.Host && strings.ToUpper(req.Request.Method) == strings.ToUpper(rr.Method) {
			return req
		}
	}
	return r
}

func (m *httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Println( fmt.Sprintf("Method:'%v'  URL:%v" , req.Method,req.URL))
	key := fmt.Sprintf("#%v_#%v_#%v", req.Host, strings.ToUpper(req.Method), req.URL.Path)
	if rsp, ok := m.mockHttpServer.reqMap[key]; ok {
		if rsp.Headers != nil {
			for k, v := range *rsp.Headers {
				resp.Header().Set(k, v)
			}
		}
		resp.WriteHeader(rsp.Status)
		resp.Write([]byte(rsp.Body))
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
				resp.WriteHeader(r.Response.Status)
				resp.Write([]byte(r.Response.Body))
				return
			}
		}
	}
	if f {
		resp.WriteHeader(404)
		resp.Write([]byte("404 Not Found"))
		return
	}
	//not pass proxy
	log.Println("")
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

func (m *MockHttpServer) start() {
	m.once.Do(func() {
		var handler httpHandler
		handler.mockHttpServer = m
		handler.allowProxyHosts = m.allowProxyHosts
		mux := http.NewServeMux()
		mux.HandleFunc("/", handler.ServeHTTP)
		go func() {
			err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", m.httpProxyPort), mux)
			if err != nil {
				panic(err)
			}
		}()
		i := 0
		for {
			_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%v", m.httpProxyPort))
			if err == nil || i > 10 {
				break
			}
			i++
			time.Sleep(time.Second)
		}
	})

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

func (m *MockHttpServer) InitMockHttpServer() *MockHttpServer {
	os.Setenv("HTTP_PROXY", fmt.Sprintf("http://127.0.0.1:%v", m.httpProxyPort))
	os.Setenv("HTTPS_PROXY", fmt.Sprintf("http://127.0.0.1:%v", m.httpProxyPort))

	//go func() {
	//	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", m.httpProxyPort), m.handler)
	//	if err != nil {
	//		panic(err)
	//	}
	//}()
	//router := mux.NewRouter()
	//httpServer := fts.NewServer(m.mockApiDir, router, &http.Server{Addr: fmt.Sprintf("0.0.0.0:%v", m.fakeHttpPort), Handler: router}, &fts.Proxy{}, false)
	//err := httpServer.Build()
	//if err != nil {
	//	panic(err)
	//}
	//httpServer.Run()

	m.start()

	return m

}

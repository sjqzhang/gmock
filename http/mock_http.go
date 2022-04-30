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
)

type MockHttpServer struct {
	fakeHttpPort    int      `json:"fake_http_port"`
	httpProxyPort   int      `json:"http_proxy_port"`
	allowProxyHosts []string `json:"allow_proxy_hosts"`
	mockApiDir      string   `json:"mock_api_dir"`
	handler         http.Handler
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
	}
	hander := &httpHandler{
		allowProxyHosts: allowProxyHosts,
		mockHttpServer:  server,
	}
	server.handler = hander

	return server

}

func (m *httpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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

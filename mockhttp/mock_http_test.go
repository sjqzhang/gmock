package mockhttp

import (
	"github.com/sjqzhang/requests"
	"testing"
)

func TestHttpServer(t *testing.T) {
	httpMock := NewMockHttpServer(12345, "./", []string{"www.baidu.com"})
	httpMock.SetReqRspHandler(func(req *Request, rsp *Response) {
		req.Method = "POST"
		req.Endpoint = "/index.html"
		req.Host = "www.baidu.com"
		req.Body = `{"name":test}`
		rsp.Body = "baidu!"
	})
	httpMock.InitMockHttpServer()


	resp, er := requests.PostJson("https://www.baidu.com/index.html", `{"name":test}`)
	//resp, err := requests.Get("http://www.baidu.com/index.html")
	if er != nil {
		t.Fail()
	}
	if resp.Text() != "baidu!" {
		t.Fail()
	}

	resp, err := requests.PostJson("http://www.baidu.com/index.html", `{"name":test}`)
	//resp, err := requests.Get("http://www.baidu.com/index.html")
	if err != nil {
		t.Fail()
	}
	if resp.Text() != "baidu!" {
		t.Fail()
	}

	httpMock.SetReqRspHandler(func(req *Request, rsp *Response) {
		req.Method = "GET"
		req.Endpoint = "/index.html"
		req.Host = "127.0.0.1:12345" //direct with http port
		rsp.Body = "baidu!"
	})

	resp, err = requests.Get("http://127.0.0.1:12345/index.html")
	if err != nil {
		t.Fail()
	}
	if resp.Text() != "baidu!" {
		t.Fail()
	}

	//
	//resp,err=requests.Get("http://www.163.com/index.html")
	//if err!=nil {
	//	t.Fail()
	//}
	//if resp.Text()!="baidu!" {
	//	t.Fail()
	//}

}

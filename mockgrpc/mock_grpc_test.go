package mockgrpc

import (
	"testing"
	"time"
)




func TestServer(t *testing.T) {
	svc:=NewMockGRPC(WithDirProtocs("/Users/junqiang.zhang/repo/go/gripmock/example/simple"))
	err:= svc.Start()
	if err==nil {
		time.Sleep(time.Second*1000)
	}
}

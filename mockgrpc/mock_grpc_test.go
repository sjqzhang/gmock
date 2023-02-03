package mockgrpc

import (
	"fmt"
	"testing"
)




func TestServer(t *testing.T) {
	svc:=NewMockGRPC(WithDirProtocs("../example/grpc"))
	err:= svc.Start()
	if err==nil {
		fmt.Println("Ok")
	} else {
		t.Fail()
	}
}

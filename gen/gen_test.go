package gen

import (
	"fmt"
	"github.com/sjqzhang/gmock/util"
	"testing"
)

func TestGen(t *testing.T) {


	a,_:=util.ParseWithPattern("func\\s+\\(\\w+\\s+(?P<Struct>[\\w\\*]+)\\)\\s(?P<Name>[\\w]+)\\(",`func (d *DSNValues) GetInt(paramName string, defaultValue int) int {`)

	tpl:=`
func Test{{.Struct}}{{.Name}}(t *testing.T) {
	Reset()
    var req req
	reqJson:="{}"
	err := json.Unmarshal([]byte(reqJson), &req)
	if err!=nil {
	  t.Fail()
	}

	resp := ctrls.{{.Struct}}Ctrl.{{.Name}}(ctx, &req)
	dtoResp := resp.(*dto.ResponseDto)
	assert.Equal(t, infra.ReturnCode(0), dtoResp.Retcode)
	
}
`
	fmt.Println(util.Render(tpl,a))
}
#! -*-encoding: utf-8 -*-
import time

# read directory files
import os
import re


def find_files(directory, pattern):
    all_files=[]
    for root, dirs, files in os.walk(directory):
        for basename in files:
            if basename.endswith(pattern):
                filename = os.path.join(root, basename)
                all_files.append(filename)
    return all_files



# parse go file content get function name

def parse_ctrl_file(file):
    results=[]
    with open(file, 'r') as f:
            lines = f.readlines()
            for line in lines:
                line=line.strip()
                if line.startswith('func') and line.find('rest.Response')>0: # find func name by feature
                    m=re.findall(r'(\w+)\)\s(\w+)[\s\S]+req\s+\*([\w+\.]+)',line)
                    if len(m)>0:
                        results.append({'file':file,'ctrl':m[0][0],'func':m[0][1],'req':m[0][2],'test':file.replace('.go','_test.go')})
    return results

# parse go test file content get function name
def parse_test_file(file):
    results=[]
    with open(file, 'r') as f:
            lines = f.readlines()
            for line in lines:
                line=line.strip()
                if line.startswith('func') and line.find('*testing.T)')>0:
                    m=re.findall(r'func\s+(\w+)',line)
                    if len(m)>0:
                        results.append({'file':file,'func':m[0]})
    return results

files=find_files('./../../mock/app','.go')


# modify test template function segment
func_tpl='''
func Test{ctrl}{func}(t *testing.T) {{
	Reset()
    var req {req}
	reqJson:=`{{}}`
	err := json.Unmarshal([]byte(reqJson), &req)
	if err!=nil {{
	  t.Fail()
	}}

	resp := ctrls.{ctrl}Ctrl.{func}(ctx, &req)
	dtoResp := resp.(*dto.ResponseDto)
	assert.Equal(t, infra.ReturnCode(0), dtoResp.Retcode)
	
}}

'''

# modify test template package segment
package_tpl='''
package app

import (
	"testing"
	"encoding/json"


)

'''



def gen_test_file(file):
    ctrls=find_files('./','go')
    for ctrl in ctrls:
        if ctrl!=file and file!='':
            continue
        parses=parse_ctrl_file(ctrl)
        funcs=[]
        for parse in parses:
            func=func_tpl.format(func=parse['func'],req=parse['req'],ctrl=parse['ctrl'])
            funcs.append(func)
        path=ctrl.replace('./','./../../mock/app/').replace('.go','_test.go')
        if not os.path.exists(path) and len(funcs)>0:
            with open(path,'w+') as fp:
                fp.write(package_tpl+'\n'.join(funcs))
        else:
            print(package_tpl+'\n'.join(funcs))


# 指定要生成单元测试的文件以 ./ 开发
gen_test_file('./xxx.go')
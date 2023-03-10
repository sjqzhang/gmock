#! -*-encoding: utf-8 -*-

# read directory files
import os
import re


def find_files(directory, pattern):
    all_files = []
    for root, dirs, files in os.walk(directory):
        for basename in files:
            if basename.endswith(pattern):
                filename = os.path.join(root, basename)
                all_files.append(filename)
    return all_files


# parse go file content get function name

def parse_ctrl_file(file):
    results = []
    with open(file, 'r') as f:
        lines = f.readlines()
        for line in lines:
            line = line.strip()
            if line.startswith('func') and line.find('*gin.Context') > 0:  # find func name by feature
                m = re.findall(r'(\w+)\)\s(\w+)[\s\S]+\*([\w+\.]+)', line)
                if len(m) > 0:
                    results.append({'file': file, 'ctrl': m[0][0], 'func': m[0][1], 'req': m[0][2],
                                    'test': file.replace('.go', '_test.go')})
    return results


# parse go test file content get function name
def parse_test_file(file):
    results = []
    with open(file, 'r') as f:
        lines = f.readlines()
        for line in lines:
            line = line.strip()
            if line.startswith('func') and line.find('*testing.T)') > 0:
                m = re.findall(r'func\s+(\w+)', line)
                if len(m) > 0:
                    results.append({'file': file, 'func': m[0]})
    return results


# modify test template function segment
func_tpl = '''
func Test{ctrl}{func}(t *testing.T) {{
	//Reset()
	var result map[string]interface{{}} //gin.Context
	reqJson := `{{}}`
	resp, err := requests.PostJson("http://127.0.0.1:8081/", reqJson)
	if err != nil {{
		t.Fail()
	}}
	if resp.R.StatusCode != 200 {{
		t.Fail()
	}}
	err = json.Unmarshal([]byte(resp.Text()), &result)
	if err != nil {{
		t.Fail()
	}}
	if util.Util.Jq(result,"code").(float64) != 200 {{
		t.Fail()
	}}

	
}}

'''

# modify test template package segment
package_tpl = '''
package app

import (
	"encoding/json"
	"github.com/sjqzhang/gmock/util"
	"github.com/sjqzhang/requests"
	"testing"


)

'''


def gen_test_file(file):
    ctrls = find_files('./', 'go')
    for ctrl in ctrls:
        if ctrl != file and file != '':
            continue
        parses = parse_ctrl_file(ctrl)
        funcs = []
        for parse in parses:
            func = func_tpl.format(func=parse['func'], req=parse['req'], ctrl=parse['ctrl'])
            funcs.append(func)
        path = ctrl.replace('./', '../mock/app/').replace('.go', '_test.go')
        if not os.path.exists(path) and len(funcs) > 0:
            with open(path, 'w+') as fp:
                fp.write(package_tpl + '\n'.join(funcs))
        else:
            print(package_tpl + '\n'.join(funcs))


# 指定要生成单元测试的文件以 ./ 开发
gen_test_file('')

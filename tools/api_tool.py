#!/usr/bin/env python3
# -*- coding:utf-8 -*-
# from flask import Flask, request, jsonify
import json
import sys
import os

txt = '''

**********  REQUEST  127.0.0.1:50982  ----->  127.0.0.1:8080  //  2023-03-22T16:09:23.07199+08:00
curl -X GET http://127.0.0.1:8080/api/admin/view/detail \
    -H 'Host: 127.0.0.1:8080' \
    -H 'User-Agent: Go-Requests 0.8' \
    -H 'Content-Type: application/json' \
    -H 'Accept-Encoding: gzip'


**********  RESPONSE  127.0.0.1:50982  <-----  127.0.0.1:8080  //  2023-03-22T16:09:23.07199+08:00 - 2023-03-22T16:09:23.079549+08:00 = 7.559ms
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 22 Mar 2023 08:09:23 GMT
Content-Length: 188

{
    "data": null,
    "message": "Invalid params.schema validate error Key: 'ViewDetailRequest.ViewId' Error:Field validation for 'ViewId' failed on the 'required' tag",
    "retcode": -1000002
}


**********  REQUEST  127.0.0.1:50982  ----->  127.0.0.1:8080  //  2023-03-22T16:09:23.079549+08:00
curl -X POST http://127.0.0.1:8080/api/admin/view/config_keys/values \
    -H 'Content-Type: application/json' \
    -H 'Accept-Encoding: gzip' \
    -H 'Host: 127.0.0.1:8080' \
    -H 'User-Agent: Go-Requests 0.8'
    -d '"{}"'

**********  RESPONSE  127.0.0.1:50982  <-----  127.0.0.1:8080  //  2023-03-22T16:09:23.079549+08:00 - 2023-03-22T16:09:23.079549+08:00 = 0s
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 22 Mar 2023 08:09:23 GMT
Content-Length: 73

{
    "data": {
        "configs": null
    },
    "message": "Success",
    "retcode": 0
}


**********  REQUEST  127.0.0.1:50982  ----->  127.0.0.1:8080  //  2023-03-22T16:09:23.079549+08:00
curl -X POST http://127.0.0.1:8080/api/admin/view/search/config_infos \
    -H 'User-Agent: Go-Requests 0.8' \
    -H 'Content-Type: application/json' \
    -H 'Accept-Encoding: gzip' \
    -H 'Host: 127.0.0.1:8080'
    -d '"{}"'

**********  RESPONSE  127.0.0.1:50982  <-----  127.0.0.1:8080  //  2023-03-22T16:09:23.079549+08:00 - 2023-03-22T16:09:23.079549+08:00 = 0s
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 22 Mar 2023 08:09:23 GMT
Content-Length: 77

{
    "data": null,
    "message": "Invalid params.scene is invalid",
    "retcode": -1000002
}


**********  REQUEST  127.0.0.1:50982  ----->  127.0.0.1:8080  //  2023-03-22T16:09:23.079549+08:00
curl -X POST http://127.0.0.1:8080/api/admin/view/search/values \
    -H 'Content-Type: application/json' \
    -H 'Accept-Encoding: gzip' \
    -H 'Host: 127.0.0.1:8080' \
    -H 'User-Agent: Go-Requests 0.8'
    -d '"{}"'

**********  RESPONSE  127.0.0.1:50982  <-----  127.0.0.1:8080  //  2023-03-22T16:09:23.079549+08:00 - 2023-03-22T16:09:23.079549+08:00 = 0s
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 22 Mar 2023 08:09:23 GMT
Content-Length: 73

{
    "data": {
        "configs": null
    },
    "message": "Success",
    "retcode": 0
}
'''

import re





def gen_testcase(req):
    func_tpl = '''
    func Test{func}(t *testing.T) {{
        //Reset()
        var result map[string]interface{{}} //gin.Context
        reqJson := `{req_body}`
        resp, err := requests.{method}("{url}", reqJson)
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
        if util.Util.Jq(result,"retcode").(float64) != 0 {{
            t.Fail()
        }}
        
    }}
    '''
    # replace host with 127.0.0.1
    req['url'] = re.sub(r'http://[^\/]+', 'http://127.0.0.1:8080', req['url'],1)
    api=re.findall(r'http://[^\/]+/([^?]+)', req['url'])
    if len(api) == 0:
        #print(req['url'])
        return ''
    apis=api[0].split('/')
    try:
        if isinstance(req['resp_body'], dict) or isinstance(req['resp_body'], list):
            req['resp_body'] = json.dumps(req['resp_body'],indent=4)
    except Exception as e:
        pass
    try:
        if isinstance(req['req_body'], dict) or isinstance(req['req_body'], list):
            req['req_body'] = json.dumps(req['req_body'],indent=4)
    except Exception as e:
        pass
    # upper apis
    if req['method'] == 'GET':
        req['method'] = 'GetJson'
    elif req['method'] == 'POST':
        req['method'] = 'PostJson'
    elif req['method'] == 'PUT':
        req['method'] = 'PutJson'
    # in req_body resp_body replace '`' with ''
    req['req_body'] = req['req_body'].replace('`','')
    req['resp_body'] = req['resp_body'].replace('`','')
    req['func'] = ''.join([x.title().replace('_','').replace('-','') for x in apis])
    return func_tpl.format(**req)


def parse_request(txt):
    uniq= os.environ.get('UNIQ','0')
    if uniq == '1':
        uniq= True
    else:
        uniq= False
    req_map = {}
    reqs = re.split(r'[\*\s]+REQUEST[^\n]+?\n', txt)
    req_list = []
    from collections import OrderedDict
    req_dict=OrderedDict()
    import uuid
    for req in reqs:
        # parse response
        resp = re.split(r'[\*\s]+RESPONSE[^\n]+?\n', req)
        if len(resp) == 2:
            req_map = {}
            req = resp[0]
            resp = resp[1]
            req_detail = re.findall(r'curl -X\s+(\w+)\s+([^\s]+)\s+', req)
            if len(req_detail) == 1:
                req_map['method'] = req_detail[0][0]
                req_map['url'] = req_detail[0][1]

                req_map['url'] = re.sub(r'http://[^\/]+', 'http://127.0.0.1:8080', req_map['url'],1)
                api=re.findall(r'http://[^\/]+/([^?]+)', req_map['url'])
                apis=api[0].split('/')
                req_map['func'] = req_map['method']+''.join([x.title().replace('_','').replace('-','') for x in apis])
                reqbody = re.findall(
                    r'-d\s+@-\s+<<\s+HTTP_DUMP_BODY_EOF\s+([\s\S]+?)\s+HTTP_DUMP_BODY_EOF|-d\s+\'([^\']+)\'', req)
                # print(reqbody)
                req_map['req_body'] = ''
                if len(reqbody) == 1:
                    if reqbody[0][0] != '':
                        req_map['req_body'] = reqbody[0][0]
                    if reqbody[0][1] != '':
                        req_map['req_body'] = reqbody[0][1]
                try:
                    req_body=json.loads( req_map['req_body'])
                    if isinstance(req_body,str):
                        req_map['req_body'] = json.loads( req_body)
                    else:
                        req_map['req_body'] = req_body
                except Exception as er:
                    pass
                lines = resp.split("\n")
                for i, v in enumerate(lines):
                    if v == '':
                        resp = "\n".join(lines[i + 1:])
                        break
                req_map['resp_body'] = resp
                try:
                    req_map['resp_body'] = json.loads(req_map['resp_body'])
                except:
                    pass
                if uniq:
                    req_dict[req_map['func']]=req_map # use newest
                else:
                    req_dict[uuid.uuid1()]=req_map
    for k,v in req_dict.items():
        req_list.append(v)
    return req_list




if __name__ == '__main__':
    if len(sys.argv) == 2:
        with open(sys.argv[1]) as f:
            txt = f.read()
            req_list = parse_request(txt)
            print(json.dumps(req_list, indent=4))
    elif len(sys.argv) == 3:
        with open(sys.argv[1]) as f:
            txt = f.read()
            req_list = parse_request(txt)
            if sys.argv[2]=='all':
                for req in req_list:
                    print(gen_testcase(req))
            else:
                for req in req_list:
                    if req['url'].find(sys.argv[2]) != -1 and sys.argv[2]!='all':
                        print(gen_testcase(req))

    else:
        print("install httpdump from https://github.com/hsiafan/httpdump")
        print("httpdump -curl=true -pretty=true -uri='/api/admin/*' -level=all -output=api.log")
        print("Usage: python3 api_tool.py api.log or python3 api_tool.py api.log all/uri pattern")

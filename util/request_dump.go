package util

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func init() {

	// load json file  to recordMap
	// read json file to recordMap
	jsBytes, err := ioutil.ReadFile("record.json")
	if err != nil {
		fmt.Println("read file error")
	} else {
		err = json.Unmarshal(jsBytes, &recordMap)
		if err != nil {
			fmt.Println("json unmarshal error")
		}
	}

	// write recordMap to json file with go routine
	go func() {
		save := func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Println("save error")
				}
			}()
			if len(recordMap) == 0 {
				return
			}
			jsBytes, err := json.MarshalIndent(recordMap, "", "  ")
			if err != nil {
				fmt.Println("json marshal error")

			}
			err = ioutil.WriteFile("record.json", jsBytes, 0644)
			if err != nil {
				fmt.Println("write file error")

			}
		}
		for {
			time.Sleep(time.Second * 10)
			save()

		}
	}()

}

type ReqRsp struct {
	ReqBody string `json:"reqBody"`
	//ReqHeader map[string]string `json:"reqHeader"`
	ReqUrl    string `json:"reqUrl"`
	ReqMethod string `json:"reqMethod"`
	RspBody   string `json:"rspBody"`
}

var recordMap = make(map[string]*ReqRsp)

var lock sync.Mutex

func Dump() gin.HandlerFunc {
	return DumpWithOptions(true, true, true, true, true, nil)
}

func DumpWithOptions(showReq bool, showResp bool, showBody bool, showHeaders bool, showCookies bool, cb func(dumpStr string)) gin.HandlerFunc {
	headerHiddenFields := make([]string, 0)
	bodyHiddenFields := make([]string, 0)

	if !showCookies {
		headerHiddenFields = append(headerHiddenFields, "cookie")
	}

	return func(ctx *gin.Context) {
		var strB strings.Builder
		var reqRsp ReqRsp
		lock.Lock()
		recordMap[fmt.Sprintf("%v,%v", ctx.Request.Method, ctx.Request.RequestURI)] = &reqRsp
		lock.Unlock()
		reqRsp.ReqMethod = ctx.Request.Method
		reqRsp.ReqUrl = ctx.Request.RequestURI

		if showReq && showHeaders {
			//dump req header
			s, err := FormatToBeautifulJson(ctx.Request.Header, headerHiddenFields)

			if err != nil {
				strB.WriteString(fmt.Sprintf("\nparse req header err \n" + err.Error()))
			} else {
				strB.WriteString("Request-Header:\n")
				strB.WriteString(string(s))
			}
		}

		if showReq && showBody {
			//dump req body
			if ctx.Request != nil && ctx.Request.Body != nil {
				buf, err := ioutil.ReadAll(ctx.Request.Body)
				if err != nil {
					strB.WriteString(fmt.Sprintf("\nread bodyCache err \n %s", err.Error()))
					goto DumpRes
				}
				rdr := ioutil.NopCloser(bytes.NewBuffer(buf))
				ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
				ctGet := ctx.Request.Header.Get("Content-Type")
				if len(buf) < 1024*1024 {
					reqRsp.ReqBody = string(buf)
				} else {
					reqRsp.ReqBody = "body too long"
				}
				ct, _, err := mime.ParseMediaType(ctGet)
				if err != nil {
					strB.WriteString(fmt.Sprintf("\ncontent_type: %s parse err \n %s", ctGet, err.Error()))
					goto DumpRes
				}

				switch ct {
				case gin.MIMEJSON:
					bts, err := ioutil.ReadAll(rdr)
					if err != nil {
						strB.WriteString(fmt.Sprintf("\nread rdr err \n %s", err.Error()))
						goto DumpRes
					}

					s, err := BeautifyJsonBytes(bts, bodyHiddenFields)
					if err != nil {
						strB.WriteString(fmt.Sprintf("\nparse req body err \n" + err.Error()))
						goto DumpRes
					}
					strB.WriteString("\nRequest-URI: " + ctx.Request.RequestURI)
					reqRsp.ReqBody = string(s)
					strB.WriteString("\nRequest-Body:\n")
					strB.WriteString(string(s))
				case gin.MIMEPOSTForm:
					bts, err := ioutil.ReadAll(rdr)
					if err != nil {
						strB.WriteString(fmt.Sprintf("\nread rdr err \n %s", err.Error()))
						goto DumpRes
					}
					val, err := url.ParseQuery(string(bts))

					s, err := FormatToBeautifulJson(val, bodyHiddenFields)
					if err != nil {
						strB.WriteString(fmt.Sprintf("\nparse req body err \n" + err.Error()))
						goto DumpRes
					}
					strB.WriteString("\nRequest-URI: " + ctx.Request.RequestURI)
					strB.WriteString("\nRequest-Body:\n")
					strB.WriteString(string(s))

				case gin.MIMEMultipartPOSTForm:

				default:
					s, e := ioutil.ReadAll(rdr)
					if e != nil {
						s = []byte("read bodyCache err")
					}
					strB.WriteString("\nRequest-URI: " + ctx.Request.RequestURI)

					strB.WriteString("\nRequest-Body:\n")
					strB.WriteString(string(s))
					goto DumpRes
				}
			}

		DumpRes:
			ctx.Writer = &bodyWriter{bodyCache: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
			ctx.Next()
		}

		if showResp && showHeaders {
			//dump res header
			sHeader, err := FormatToBeautifulJson(ctx.Writer.Header(), headerHiddenFields)
			if err != nil {
				strB.WriteString(fmt.Sprintf("\nparse res header err \n" + err.Error()))
			} else {
				strB.WriteString("\nResponse-Header:\n")
				strB.WriteString(string(sHeader))
			}
		}

		if showResp && showBody {
			var body []byte
			bw, ok := ctx.Writer.(*bodyWriter)
			if !ok {

				strB.WriteString("\nbodyWriter was override , can not read bodyCache")
				goto End

			} else {
				body = bw.bodyCache.Bytes()
				if len(body) < 1024*1024 {
					reqRsp.RspBody = string(body)
				} else {
					reqRsp.RspBody = "body too long"
				}
			}

			//dump res body
			if bodyAllowedForStatus(ctx.Writer.Status()) && len(body) > 0 {
				ctGet := ctx.Writer.Header().Get("Content-Type")
				ct, _, err := mime.ParseMediaType(ctGet)
				if err != nil {
					strB.WriteString(fmt.Sprintf("\ncontent-type: %s parse  err \n %s", ctGet, err.Error()))
					goto End
				}
				switch ct {
				case gin.MIMEJSON:

					s, err := BeautifyJsonBytes(body, bodyHiddenFields)
					if err != nil {
						strB.WriteString(fmt.Sprintf("\nparse bodyCache err \n" + err.Error()))
						goto End
					}
					strB.WriteString("\nResponse-Body:\n")

					strB.WriteString(string(s))
				case gin.MIMEHTML:
				default:
				}
			}
		}

	End:
		if cb != nil {
			cb(strB.String())
		} else {
			fmt.Println(strB.String())
		}
	}
}

type bodyWriter struct {
	gin.ResponseWriter
	bodyCache *bytes.Buffer
}

//rewrite Write()
func (w bodyWriter) Write(b []byte) (int, error) {
	w.bodyCache.Write(b)
	return w.ResponseWriter.Write(b)
}

// bodyAllowedForStatus is a copy of http.bodyAllowedForStatus non-exported function.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == http.StatusNoContent:
		return false
	case status == http.StatusNotModified:
		return false
	}
	return true
}

var StringMaxLength = 0
var Newline = "\n"
var Indent = 4

func BeautifyJsonBytes(data []byte, hiddenFields []string) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	v = removeHiddenFields(v, hiddenFields)

	return []byte(format(v, 1)), nil
}

//transfer v to beautified json bytes
func FormatToBeautifulJson(v interface{}, hiddenFields []string) ([]byte, error) {

	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return BeautifyJsonBytes(data, hiddenFields)
}

func format(v interface{}, depth int) string {
	switch val := v.(type) {
	case string:
		return formatString(val)
	case float64:
		return fmt.Sprint(strconv.FormatFloat(val, 'f', -1, 64))
	case bool:
		return fmt.Sprint(strconv.FormatBool(val))
	case nil:
		return fmt.Sprint("null")
	case map[string]interface{}:
		return formatMap(val, depth)
	case []interface{}:
		return formatArray(val, depth)
	}

	return ""
}

func formatString(s string) string {
	r := []rune(s)
	if StringMaxLength != 0 && len(r) >= StringMaxLength {
		s = string(r[0:StringMaxLength]) + "..."
	}

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(s)
	s = string(buf.Bytes())
	s = strings.TrimSuffix(s, "\n")

	return fmt.Sprint(s)
}

func formatMap(m map[string]interface{}, depth int) string {
	if len(m) == 0 {
		return "{}"
	}

	currentIndent := generateIndent(depth - 1)
	nextIndent := generateIndent(depth)
	rows := []string{}
	keys := []string{}

	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		val := m[key]
		k := fmt.Sprintf(`"%s"`, key)
		v := format(val, depth+1)

		valueIndent := " "
		if Newline == "" {
			valueIndent = ""
		}
		row := fmt.Sprintf("%s%s:%s%s", nextIndent, k, valueIndent, v)
		rows = append(rows, row)
	}

	return fmt.Sprintf("{%s%s%s%s}", Newline, strings.Join(rows, ","+Newline), Newline, currentIndent)
}

func formatArray(a []interface{}, depth int) string {
	if len(a) == 0 {
		return "[]"
	}

	currentIndent := generateIndent(depth - 1)
	nextIndent := generateIndent(depth)
	rows := []string{}

	for _, val := range a {
		c := format(val, depth+1)
		row := nextIndent + c
		rows = append(rows, row)
	}
	return fmt.Sprintf("[%s%s%s%s]", Newline, strings.Join(rows, ","+Newline), Newline, currentIndent)
}

func generateIndent(depth int) string {
	return strings.Repeat(" ", Indent*depth)
}

func removeHiddenFields(v interface{}, hiddenFields []string) interface{} {
	if _, ok := v.(map[string]interface{}); !ok {
		return v
	}

	m := v.(map[string]interface{})

	// case insensitive key deletion
	for _, hiddenField := range hiddenFields {
		for k := range m {
			if strings.ToLower(k) == strings.ToLower(hiddenField) {
				delete(m, k)
			}
		}
	}

	return m
}

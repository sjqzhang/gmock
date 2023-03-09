package util

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDSNParser(t *testing.T) {
	dsn := "mysql://root:123456@x@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"

	d, err := Parse(dsn)

	if err != nil {
		panic(err)
	}

	fmt.Println(d.DSN(true))
	if d.DSN(true) != dsn {
		t.Fail()
	}

}


func TestGinMiddleware(t *testing.T) {
	// start gin server for test
	r := gin.Default()
	r.Use(DumpWithOptions(true, true,true,false,false, func(dumpStr string) {
		if dumpStr != "\nResponse-Body:\n{\n    \"message\": \"pong\"\n}" {
			t.Fail()
		}
	}))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// mock http request for test
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	// test
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fail()
	}




}
package util

import (
	"fmt"
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

package util

import (
	"fmt"
	"github.com/sjqzhang/goutil"
	"log"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type logger struct {
	tag string
	log *log.Logger
}

func NewLogger(tag string) *logger {
	return &logger{tag: tag, log: log.New(os.Stdout, fmt.Sprintf("[%v] ", tag), log.LstdFlags)}
}
func (l *logger) SetTag(tag string) {
	l.tag = tag
}
func (l *logger) Log(msg interface{}) {
	l.log.Println("\u001B[32m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *logger) Warn(msg interface{}) {
	l.log.Println("\u001B[33m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *logger) Error(msg interface{}) {

	l.log.Println("\u001B[31m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *logger) Panic(msg interface{}) {
	panic("\u001B[31m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}

var Logger *logger = NewLogger("default")

func Recover() {
	if err := recover(); err != nil {
		_, file, line, ok := runtime.Caller(3)
		if ok {
			errMsg := fmt.Sprintf("[%s] panic file:[%s:%v] recovered:\n%s\n%s", "gmock", file, line, err, string(debug.Stack()))
			Logger.Error(errMsg)
		}
	}
}

var Util *goutil.Common = &goutil.Common{}

func CheckPortIsReady(addr string) (bool, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	return true, nil
}

func Exec(cmd string) (string, int) {
	if runtime.GOOS == "windows" {
		return Util.Exec([]string{"cmd", "/C", cmd}, 3600)
	}
	return Util.Exec([]string{"sh", "-c", cmd}, 3600)
}

type DSN struct {
	url *url.URL
	*DSNValues
	isWrapTcp bool
}

// parses dsn string and returns DSN instance
func Parse(dsn string) (*DSN, error) {
	reg := regexp.MustCompile(`tcp\(.*?\)`) //uniform url format
	isWrapTcp := false
	if m := reg.FindStringSubmatch(dsn); len(m) > 0 {
		match := m[0]
		match = strings.TrimPrefix(match, "tcp(")
		match = strings.TrimSuffix(match, ")")
		dsn = reg.ReplaceAllString(dsn, match)
		isWrapTcp = true
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	d := DSN{
		parsed,
		&DSNValues{parsed.Query()}, isWrapTcp,
	}
	return &d, nil
}


func CollectFieldNames(t reflect.Type,m map[string]struct{},prefix string) {
	// Return if not struct or pointer to struct.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	// Iterate through fields collecting names in map.
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		m[prefix+ sf.Name] = struct{}{}

		// Recurse into anonymous fields.
		if sf.Anonymous {
			CollectFieldNames(sf.Type, m,prefix+sf.Name+",")
		}
	}
}

// Parses query and returns dsn values
func ParseQuery(query string) (*DSNValues, error) {
	parsed, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	return &DSNValues{parsed}, nil
}

// returns DSNValues from url.Values
func NewValues(query url.Values) (*DSNValues, error) {
	return &DSNValues{query}, nil
}

// return Host
func (d *DSN) DSN(withSchema bool) string {
	schema := ""
	if withSchema {
		schema = d.url.Scheme + "://"
	}
	if d.isWrapTcp {
		
		return schema + d.Username() + ":" + d.Password() + "@tcp(" + d.url.Host + ")" + d.url.Path + "?" + d.url.RawQuery
	} else {
		return schema + d.Username() + ":" + d.Password() + "@" + d.url.Host + d.url.Path + "?" + d.url.RawQuery
	}
}

// return Host
func (d *DSN) HostWithPort() string {
	return d.url.Host
}

// return Host
func (d *DSN) Host() string {
	return strings.Split(d.url.Host, ":")[0]
}

// return Host
func (d *DSN) Port() string {
	hp := strings.Split(d.url.Host, ":")
	if len(hp) == 2 {
		return hp[1]
	} else {
		return ""
	}
}

// return Scheme
func (d *DSN) Scheme() string {
	return d.url.Scheme
}

// returns path
func (d *DSN) Path() string {
	return d.url.Path
}

// returns path
func (d *DSN) DatabaseName() string {
	return strings.Replace(d.url.Path, "/", "", -1)
}

// returns path
func (d *DSN) SetDatabaseName(dbName string) {
	d.url.Path = "/" + dbName
}

// returns user
func (d *DSN) User() *url.Userinfo {
	return d.url.User
}

// returns Username
func (d *DSN) Username() string {
	return d.url.User.Username()
}

// returns Username
func (d *DSN) Password() string {
	v, ok := d.url.User.Password()
	if ok {
		return v
	} else {
		return ""
	}
}

// DSN Values
type DSNValues struct {
	url.Values
}

// returns int value
func (d *DSNValues) GetInt(paramName string, defaultValue int) int {
	value := d.Get(paramName)
	if i, err := strconv.Atoi(value); err == nil {
		return i
	} else {
		return defaultValue
	}
}

// returns string value
func (d *DSNValues) GetString(paramName string, defaultValue string) string {
	value := d.Get(paramName)
	if value == "" {
		return defaultValue
	} else {
		return value
	}
}

// returns string value
func (d *DSNValues) GetBool(paramName string, defaultValue bool) bool {
	value := strings.ToLower(d.Get(paramName))
	if value == "true" || value == "1" {
		return true
	} else if value == "0" || value == "false" {
		return false
	} else {
		return defaultValue
	}
}

// returns string value
func (d *DSNValues) GetSeconds(paramName string, defaultValue time.Duration) time.Duration {
	if i, err := strconv.Atoi(d.Get(paramName)); err == nil {
		return time.Duration(i) * time.Second
	} else {
		return defaultValue
	}
}

package mockdb

import (
	"database/sql"
	"fmt"
	"github.com/sjqzhang/gmock/util"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type MockGORMV2 struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *gorm.DB
	dbType            string
	dsn               string
	models            []interface{}
	util              *util.DBUtil
	resetHandler      func(resetHandler *MockGORMV2)
	schema            string
}

func NewMockGORMV2(pathToSqlFileName string, resetHandler func(orm *MockGORMV2)) *MockGORMV2 {
	mock := MockGORMV2{
		pathToSqlFileName: pathToSqlFileName,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
	}
	var err error
	var db *gorm.DB
	ns := schema.NamingStrategy{
		SingularTable: true,
	}

	mock.util = util.NewDBUtil()
	if DBType == "mysql" {
		for i := 63306; i < 63400; i++ {
			_, e := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%v", i))
			if e == nil {
				continue
			}
			mock.util.RunMySQLServer("mock", i, false)
			time.Sleep(time.Second)
			mock.dsn= fmt.Sprintf("root:root@tcp(127.0.0.1:%v)/mock?charset=utf8&parseTime=True&loc=Local", i)
			mock.dbType="mysql"
			db, err = gorm.Open(mysql.Open(mock.dsn), &gorm.Config{NamingStrategy: ns})
			break
		}

	} else {
		mock.dbType = "sqlite3"
		mock.dsn = "file::memory:?cache=shared"
		db, err = gorm.Open(sqlite.Open(mock.dsn), &gorm.Config{
			NamingStrategy: ns,
		})
	}
	if err != nil {
		panic(err)
	}
	mock.db = db
	return &mock
}

func renew2() *gorm.DB {
	var err error
	ns := schema.NamingStrategy{
		SingularTable: true,
	}
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		NamingStrategy: ns,
	})
	if err != nil {
		panic(err)
	}
	return db
}

// ResetAndInit 初始化数据库及表数据
func (m *MockGORMV2) ResetAndInit() {
	//m.db = renew2()

	m.dropTables()
	m.initModels()
	m.initSQL()
	if m.resetHandler != nil {
		m.resetHandler(m)
	}
}

// GetGormDB 获取Gorm实例
func (m *MockGORMV2) GetGormDB() *gorm.DB {
	return m.db
}
func (m *MockGORMV2) dropTables() {
	for _, model := range m.models {
		m.db.Migrator().DropTable(model)
	}
}
func (m *MockGORMV2) GetDSN() (dbType string, dsn string) {
	dbType = m.dbType
	dsn = m.dsn
	return
}


func (m *MockGORMV2) GetDBUtil() *util.DBUtil {
	return m.util
}

func (m *MockGORMV2) InitSchemas(sqlSchema string) {
	m.schema = sqlSchema
}

// GetSqlDB  获取*sql.DB实例
func (m *MockGORMV2) GetSqlDB() *sql.DB {
	db, err := m.db.DB()
	if err != nil {
		return nil
	}
	return db
}

// RegisterModels 注册模型
func (m *MockGORMV2) RegisterModels(models ...interface{}) {
	if len(models) > 0 {
		for _, model := range models {
			mv := reflect.ValueOf(model)
			mt := reflect.TypeOf(model)
			if mt.Kind() != reflect.Ptr || reflect.TypeOf(mv.Interface()).Kind() != reflect.Struct {
				m.models = append(m.models, model)
			} else {
				logger.Panic(fmt.Sprintf("model should be struct prt"))
			}
		}
	}
}

// InitModels init table schema in db instance
func (m *MockGORMV2) initModels() {
	if m.db == nil {
		panic("warning: call ResetAndInit func first!!!!!")
	}
	for _, model := range m.models {
		err := m.db.AutoMigrate(model)
		if err != nil {
			panic(err)
		}
	}
}
func (m *MockGORMV2) initSQL() {
	if m.schema != "" {
		sqls := m.parseMockSQL(m.schema)
		for _, sql := range sqls {
			err := m.db.Exec(sql).Error
			if err != nil {
				logger.Error(sql)
				panic(err)
			}
		}
	}
	for _, filePath := range getFilesBySuffix(m.pathToSqlFileName, "sql") {
		sqlText := m.readMockSQl(filePath)
		sqls := m.parseMockSQL(sqlText)
		for _, sqlStr := range sqls {
			err := m.db.Exec(sqlStr).Error
			if err != nil {
				logger.Error(filePath)
				panic(err)
			}
		}
		logger.Log(fmt.Sprintf("sql file %v is loaded", filePath))
	}
}

// ReadMockSQl read sql file to string
func (m *MockGORMV2) readMockSQl(filePath string) string {
	if _, err := os.Stat(filePath); err != nil {
		logger.Error(err)
		return ""
	}
	fp, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(fp)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// parseMockSQL parse sql text to []string
func (m *MockGORMV2) parseMockSQL(sqlText string) []string {
	reg := regexp.MustCompile(`[\r\n]+`)
	linses := reg.Split(sqlText, -1)
	var tmp []string
	var sqls []string
	for _, line := range linses {
		tmp = append(tmp, line)
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			if len(tmp) > 0 {
				sqls = append(sqls, strings.Join(tmp, "\n"))
			}
			tmp = []string{}
		}

	}
	return sqls
}

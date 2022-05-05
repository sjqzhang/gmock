package mockdb

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type MockGORMV2 struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *gorm.DB
	models            []interface{}
	resetHandler      func(resetHandler *MockGORMV2)
}

func NewMockGORMV2(pathToSqlFileName string, resetHandler func(orm *MockGORMV2)) *MockGORMV2 {
	db := renew2()
	return &MockGORMV2{
		pathToSqlFileName: pathToSqlFileName,
		db:                db,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
	}
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
	m.db = renew2()
	if m.resetHandler!=nil {
		m.resetHandler(m)
	}
	m.initModels()
	m.initSQL()
}

// GetGormDB 获取Gorm实例
func (m *MockGORMV2) GetGormDB() *gorm.DB {
	return m.db
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
				log.Panic(fmt.Sprintf("model should be struct prt"))
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
	for _, filePath := range getFilesBySuffix(m.pathToSqlFileName, "sql") {
		sqlText := m.readMockSQl(filePath)
		sqls := m.parseMockSQL(sqlText)
		for _, sqlStr := range sqls {
			err := m.db.Exec(sqlStr).Error
			if err != nil {
				log.Print(filePath)
				panic(err)
			}
		}
		log.Printf("sql file %v is loaded", filePath)
	}
}

// ReadMockSQl read sql file to string
func (m *MockGORMV2) readMockSQl(filePath string) string {
	if _, err := os.Stat(filePath); err != nil {
		log.Print(err)
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

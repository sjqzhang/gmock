package db

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

type MockDBV2 struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *gorm.DB
	models            []interface{}
}

func NewMockDBV2(pathToSqlFileName string) *MockDBV2 {
	db := renew2()
	return &MockDBV2{
		pathToSqlFileName: pathToSqlFileName,
		db:                db,
		models:            make([]interface{}, 0),
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

// ResetAndInit reset db instance
func (m *MockDBV2) ResetAndInit() {
	m.db = renew2()
	m.initModels()
	m.initSQL()
}

func (m *MockDBV2) GetGormDB() *gorm.DB {
	return m.db
}
func (m *MockDBV2) GetSqlDB() (*sql.DB, error) {
	return m.db.DB()
}

func (m *MockDBV2) RegisterModels(models ...interface{}) {
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
func (m *MockDBV2) initModels() {
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
func (m *MockDBV2) initSQL() {
	sqlText := m.readMockSQl()
	sqls := m.parseMockSQL(sqlText)
	for _, sql := range sqls {
		err := m.db.Exec(sql).Error
		if err != nil {
			panic(err)
		}
	}
}

// ReadMockSQl read sql file to string
func (m *MockDBV2) readMockSQl() string {
	_, err := os.Stat(m.pathToSqlFileName)
	if err != nil {
		log.Printf("(warning)sql file %s not found", m.pathToSqlFileName)
		return ""
	}
	fp, err := os.Open(m.pathToSqlFileName)
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
func (m *MockDBV2) parseMockSQL(sqlText string) []string {
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

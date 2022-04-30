package db

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type MockDB struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *gorm.DB
	models            []interface{}
}

func NewMockDB(pathToSqlFileName string) *MockDB {
	db := renew()
	return &MockDB{
		pathToSqlFileName: pathToSqlFileName,
		db:                db,
		models:            make([]interface{}, 0),
	}
}

func renew() *gorm.DB {
	var err error
	db, err := gorm.Open("sqlite3", ":memory:")
	db.SingularTable(true)
	if err != nil {
		panic(err)
	}
	return db
}

// Reset reset db instance
func (m *MockDB) Reset() {
	m.db = renew()
	m.initModels()
	m.initSQL()
}

func (m *MockDB) GetGormDB() *gorm.DB {
	return m.db
}
func (m *MockDB) GetSqlDB() *sql.DB {
	return m.db.DB()
}

func (m *MockDB) RegisterModels(models ...interface{}) {
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
func (m *MockDB) initModels() {
	if m.db == nil {
		panic("warning: call Reset func first!!!!!")
	}
	for _, model := range m.models {
		err := m.db.AutoMigrate(model).Error
		if err != nil {
			panic(err)
		}
	}
}
func (m *MockDB) initSQL() {
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
func (m *MockDB) readMockSQl() string {
	_ = sqlite3.SQLITE_COPY
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
func (m *MockDB) parseMockSQL(sqlText string) []string {
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

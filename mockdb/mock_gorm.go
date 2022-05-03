package mockdb

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

type MockGORM struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *gorm.DB
	models            []interface{}
}

func NewMockGORM(pathToSqlFileName string) *MockGORM {
	db := renew()
	return &MockGORM{
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

func getFilesBySuffix(dir string, suffix string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%w: error finding imposters", err)
		}
		filename := info.Name()
		if !info.IsDir() {
			if strings.HasSuffix(filename, suffix) {
				files = append(files, path)
			}
		}
		return nil
	})
	return files
}

// ResetAndInit 初始化数据库及表数据
func (m *MockGORM) ResetAndInit() {
	m.db = renew()
	m.initModels()
	m.initSQL()
}
// GetGormDB 获取Gorm实例
func (m *MockGORM) GetGormDB() *gorm.DB {
	return m.db
}
// GetSqlDB  获取*sql.DB实例
func (m *MockGORM) GetSqlDB() *sql.DB {
	return m.db.DB()
}
// RegisterModels 注册模型
func (m *MockGORM) RegisterModels(models ...interface{}) {
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
func (m *MockGORM) initModels() {
	if m.db == nil {
		panic("warning: call ResetAndInit func first!!!!!")
	}
	for _, model := range m.models {
		err := m.db.AutoMigrate(model).Error
		if err != nil {
			panic(err)
		}
	}
}
func (m *MockGORM) initSQL() {
	for _, filePath := range getFilesBySuffix(m.pathToSqlFileName, "sql") {
		sqlText := m.readMockSQl(filePath)
		sqls := m.parseMockSQL(sqlText)
		for _, sql := range sqls {
			err := m.db.Exec(sql).Error
			if err != nil {
				log.Print(filePath)
				panic(err)
			}
		}
		log.Printf("sql file %v is loaded", filePath)
	}
}

// ReadMockSQl read sql file to string
func (m *MockGORM) readMockSQl(filePath string) string {
	_ = sqlite3.SQLITE_COPY
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
func (m *MockGORM) parseMockSQL(sqlText string) []string {
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

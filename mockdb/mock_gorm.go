package mockdb

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/mattn/go-sqlite3"
	"github.com/sjqzhang/gmock/util"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var DBType string = "sqlite3"

type MockGORM struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *gorm.DB
	dbType            string
	dsn               string
	schema            string // just fix:github.com/jinzhu/gorm   "fix tables"
	util              *util.DBUtil
	models            []interface{}
	resetHandler      func(orm *MockGORM)
}

func NewMockGORM(pathToSqlFileName string, resetHandler func(orm *MockGORM)) *MockGORM {
	mock := MockGORM{
		pathToSqlFileName: pathToSqlFileName,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
	}
	var err error
	var db *gorm.DB
	mock.util = util.NewDBUtil()
	if DBType == "mysql" {
		for i := 63306; i < 63400; i++ {
			_, e := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%v", i))
			if e == nil {
				continue
			}
			mock.util.RunMySQLServer("mock", i, false)
			time.Sleep(time.Second)
			mock.dsn = fmt.Sprintf("root:root@tcp(127.0.0.1:%v)/mock?charset=utf8&parseTime=True&loc=Local", i)
			mock.dbType = "mysql"
			db, err = gorm.Open("mysql", mock.dsn)
			break
		}
	} else {
		mock.dbType = "sqlite3"
		mock.dsn = ":memory:"
		db, err = gorm.Open("sqlite3", ":memory:")
	}
	db.SingularTable(true)
	if err != nil {
		panic(err)
	}
	mock.db = db
	return &mock

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
	//m.db = renew()

	m.dropTables()
	if m.dbType != "mysql" {
		m.initModels()
	}
	m.initSQL()
	if m.resetHandler != nil {
		m.resetHandler(m)
	}
}

//func (m *MockGORM) GetTableNames() []string {
//	s := "SELECT name FROM sqlite_master where type='table' order by name"
//	rows, err := m.GetSqlDB().Query(s)
//	var tableNames []string
//	if err == nil {
//		for rows.Next() {
//			var name string
//			err = rows.Scan(&name)
//			if err == nil {
//				tableNames = append(tableNames, name)
//			}
//		}
//	}
//	return tableNames
//}

func (m *MockGORM) dropTables() {
	if m.dbType == "mysql" {
		rows, err := m.db.Raw("show tables").Rows()
		if err == nil {
			for rows.Next() {
				var name string
				err = rows.Scan(&name)
				if err == nil {
					err = m.db.DropTable(name).Error
					if err != nil {
						log.Print(err)
					}
				}
			}

		}

	} else {
		for _, model := range m.models {
			m.db.DropTableIfExists(model)
		}
	}
}

// GetGormDB 获取Gorm实例
func (m *MockGORM) GetGormDB() *gorm.DB {
	return m.db
}

// GetSqlDB  获取*sql.DB实例
func (m *MockGORM) GetSqlDB() *sql.DB {
	return m.db.DB()
}

// InitSchemas  为了兼容github.com/jinzhu/gorm mysql的bug 特殊处理的
func (m *MockGORM) InitSchemas(sqlSchema string) {
	m.schema = sqlSchema
}

func (m *MockGORM) GetDSN() (dbType string, dsn string) {
	dbType = m.dbType
	dsn = m.dsn
	return
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
	if m.schema != "" {
		sqls := m.parseMockSQL(m.schema)
		for _, sql := range sqls {
			err := m.db.Exec(sql).Error
			if err != nil {
				log.Print(sql)
				panic(err)
			}
		}
	}
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

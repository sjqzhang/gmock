package mockdb

import (
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"github.com/sjqzhang/gmock/util"
	"io/ioutil"
	"log"
	"net"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
	"xorm.io/xorm"
)

type MockXORM struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	engine            *xorm.Engine
	//engine *xorm.Engine
	models       []interface{}
	util         *util.DBUtil
	dbType       string
	dsn          string
	resetHandler func(orm *MockXORM)
	schema       string
}

func NewMockXORM(pathToSqlFileName string, resetHandler func(orm *MockXORM)) *MockXORM {
	var db *xorm.Engine
	var err error
	mock := MockXORM{
		pathToSqlFileName: pathToSqlFileName,
		engine:            db,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
	}
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
			db, err = xorm.NewEngine(mock.dbType, mock.dsn)
			break
		}
	} else {
		mock.dbType = "sqlite3"
		mock.dsn = ":memory:"
		db, err = xorm.NewEngine("sqlite3", ":memory:")
	}
	if err != nil {
		panic(err)
	}
	mock.engine = db
	return &mock
}

//func renewEngine() *xorm.Engine {
//	var err error
//	var engine *xorm.Engine
//	engine, err = xorm.NewEngine("sqlite3", ":memory:")
//	//engine.SingularTable(true)
//	if err != nil {
//		panic(err)
//	}
//	return engine
//}

//
//func getFilesBySuffix(dir string, suffix string) []string {
//	var files []string
//	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
//		if err != nil {
//			return fmt.Errorf("%w: error finding imposters", err)
//		}
//		filename := info.Name()
//		if !info.IsDir() {
//			if strings.HasSuffix(filename, suffix) {
//				files = append(files, path)
//			}
//		}
//		return nil
//	})
//	return files
//}

// ResetAndInit 初始化数据库及表数据
func (m *MockXORM) ResetAndInit() {
	//m.engine = renewEngine()

	m.initModels()
	m.initSQL()
	if m.resetHandler != nil {
		m.resetHandler(m)
	}
}

func (m *MockXORM) dropTables() {

	m.engine.DropTables(m.models)

}

func (m *MockXORM) InitSchemas(sqlSchema string) {
	m.schema = sqlSchema
}

//GetXORMEngine 获取 *xorm.Engine实例
func (m *MockXORM) GetXORMEngine() *xorm.Engine {
	return m.engine
}

// GetSqlDB  获取*sql.DB实例
func (m *MockXORM) GetSqlDB() *sql.DB {
	return m.engine.DB().DB
}

func (m *MockXORM) GetDSN() (dbType string, dsn string) {
	dbType = m.dbType
	dsn = m.dsn
	return
}

// RegisterModels 注册模型
func (m *MockXORM) RegisterModels(models ...interface{}) {
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

// InitModels init table schema in engine instance
func (m *MockXORM) initModels() {
	if m.engine == nil {
		panic("warning: call ResetAndInit func first!!!!!")
	}
	for _, model := range m.models {
		err := m.engine.Sync(model)
		if err != nil {
			panic(err)
		}
	}
}
func (m *MockXORM) initSQL() {
	if m.schema != "" {
		sqls := m.parseMockSQL(m.schema)
		for _, sql := range sqls {
			_, err := m.engine.Exec(sql)
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
			_, err := m.engine.Exec(sql)
			if err != nil {
				log.Print(filePath)
				panic(err)
			}
		}
		log.Printf("sql file %v is loaded", filePath)
	}
}

// ReadMockSQl read sql file to string
func (m *MockXORM) readMockSQl(filePath string) string {
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
func (m *MockXORM) parseMockSQL(sqlText string) []string {
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

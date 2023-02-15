package mockdb

import (
	"context"
	"database/sql"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/mattn/go-sqlite3"
	"github.com/sjqzhang/gmock/util"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
	"xorm.io/xorm"
	"xorm.io/xorm/contexts"
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
	once         sync.Once
	hook         *hook
	//mock         *MockGORM
	recorderSQLDB *sql.DB
	recorder      map[string]mapset.Set
	schema        string
}

func NewXORMFromDSN(pathToSqlFileName string, dbType string, dsn string) *MockXORM {
	var db *xorm.Engine
	var err error
	mock := MockXORM{
		pathToSqlFileName: pathToSqlFileName,
		engine:            db,
		models:            make([]interface{}, 0),
		//resetHandler:      resetHandler,
		recorder: make(map[string]mapset.Set),
		once:     sync.Once{},
	}
	db, err = xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return &mock
}

func NewMockXORM(pathToSqlFileName string, resetHandler func(orm *MockXORM)) *MockXORM {
	var db *xorm.Engine
	var err error
	mock := MockXORM{
		pathToSqlFileName: pathToSqlFileName,
		engine:            db,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
		recorder:          make(map[string]mapset.Set),
		once:              sync.Once{},
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

//func (m *MockXORM) SetXORMEngine(engine *xorm.Engine) {
//	m.engine = engine
//}

// GetSqlDB  获取*sql.DB实例
func (m *MockXORM) GetSqlDB() *sql.DB {
	return m.engine.DB().DB
}

func (m *MockXORM) GetDSN() (dbType string, dsn string) {
	dbType = m.dbType
	dsn = m.dsn
	return
}

func (m *MockXORM) GetDBUtil() *util.DBUtil {
	return m.util
}

type hook struct {
	m *MockXORM
}

func (h *hook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {

	return c.Ctx, nil
}

func (h *hook) AfterProcess(c *contexts.ContextHook) error {

	sql := strings.TrimSpace(strings.ToUpper(c.SQL))
	if strings.HasPrefix(sql, "SELECT") {
		h.m.GetDBUtil().DoRecordQueryTableIds(h.m.GetSqlDB(), h.m.recorder, c.SQL, c.Args)
	}
	return nil
}

func (m *MockXORM) DumpRecorderInfo() map[string][]string {
	result := make(map[string][]string)
	for tableName, set := range m.recorder {
		var ids []string
		for id := range set.Iter() {
			ids = append(ids, fmt.Sprintf("%v", id))
		}
		if len(ids) > 0 {
			//sqls = append(sqls, fmt.Sprintf("select * from `%v` where id in (%v)", tableName, strings.Join(ids, ",")))
			result[tableName] = ids
		}

	}
	return result
}

func (m *MockXORM) SaveRecordToFile(dir string) {
	m.util.SaveRecordToFile(dir, m.util.DumpFromRecordInfo(m.recorderSQLDB, m.DumpRecorderInfo()), false)
}

func (m *MockXORM) SaveRecordToFileAuto(dir string) {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			m.util.SaveRecordToFile(dir, m.util.DumpFromRecordInfo(m.recorderSQLDB, m.DumpRecorderInfo()), true)
		}
	}()
}

func (m *MockXORM) DoRecord(scope *xorm.Engine) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("%v", err))
		}
	}()
	m.once.Do(func() {
		scope.DriverName()
		m.hook = &hook{m: m}
		//m.mock = NewMockGORM("", nil)
		//db, err := gorm.Open(scope.DriverName(), scope.DataSourceName())
		//if err != nil {
		//	logger.Error(err)
		//	panic(err)
		//}
		////m.mock.SetGormDB(db)
		//db.Callback().Query().After("gorm:after").Register("xxx:xxxx", func(scope *gorm.Scope) {
		//	m.mock.DoRecord(scope)
		//})
		//db.SingularTable(true)
		//m.SetXORMEngine(scope)
		if m.recorderSQLDB == nil {
			m.recorderSQLDB = scope.DB().DB
		}
		scope.DB().AddHook(m.hook)
	})

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
				logger.Panic(fmt.Sprintf("model should be struct prt"))
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
				logger.Error(sql)
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
				logger.Error(filePath)
				panic(err)
			}
		}
		logger.Log(fmt.Sprintf("sql file %v is loaded", filePath))
	}
}

// ReadMockSQl read sql file to string
func (m *MockXORM) readMockSQl(filePath string) string {
	_ = sqlite3.SQLITE_COPY
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

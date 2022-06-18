package mockdb

import (
	"database/sql"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	_ "github.com/go-sql-driver/mysql"
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
	"sync"
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
	//dbRecorder        *gorm.DB
	//onceRecorder      sync.Once
	//dumper            *xorm.Engine
	recorder      map[string]mapset.Set
	recorderSQLDB *sql.DB
	recordLock    sync.Mutex
}
type Logger struct {
	tag string
	log *log.Logger
}

func NewLogger(tag string) *Logger {
	return &Logger{tag: tag, log: log.New(os.Stdout, fmt.Sprintf("[%v] ", tag), log.LstdFlags)}
}

func (l *Logger) Log(msg interface{}) {

	l.log.Println("\u001B[32m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *Logger) Warn(msg interface{}) {
	l.log.Println("\u001B[33m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *Logger) Error(msg interface{}) {

	l.log.Println("\u001B[31m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}
func (l *Logger) Panic(msg interface{}) {
	panic("\u001B[31m" + fmt.Sprintf("%v", msg) + "\u001B[0m")
}

var logger = NewLogger("gmock.mockdb")

func NewMockGORM(pathToSqlFileName string, resetHandler func(orm *MockGORM)) *MockGORM {

	mock := MockGORM{
		pathToSqlFileName: pathToSqlFileName,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
		recorder:          make(map[string]mapset.Set),
		recordLock:        sync.Mutex{},
		//onceRecorder:      sync.Once{},
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
	//if err := m.db.Commit().Error; err != nil {
	//	logger.Error(err)
	//}
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
						logger.Error(err)
					}
				}
			}

		}

	} else {
		for _, model := range m.models {
			err := m.db.DropTableIfExists(model).Error
			if err != nil {
				logger.Error(err)
			}
		}
	}
}

// GetGormDB 获取Gorm实例
func (m *MockGORM) GetGormDB() *gorm.DB {
	return m.db
}

// GetGormDB 获取Gorm实例
//func (m *MockGORM) SetGormDB(db *gorm.DB) {
//	m.db = db
//}

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

func (m *MockGORM) GetDBUtil() *util.DBUtil {
	return m.util
}

func (m *MockGORM) DumpRecorderInfo() map[string][]string {
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

//func (m *MockGORM) Dump(w io.Writer) {
//	if m.dumper == nil {
//		logger.Error("must be call DoRecord first")
//		return
//	}
//	m.dumper.DumpAll(w)
//	m.dumper.Close()
//	m.dbRecorder.Close()
//	os.Remove(".mock.db")
//}

//func (m *MockGORM) DoRecord(scope *gorm.Scope) {
//
//	m.onceRecorder.Do(func() {
//		var err error
//		if m.dbType == "mysql" {
//			t, dsn := m.GetDSN()
//			m.dbRecorder, err = gorm.Open(t, dsn)
//		} else {
//			m.dbRecorder, err = gorm.Open("sqlite3", ".mock.db")
//		}
//		m.dbRecorder.SingularTable(true)
//		m.dumper, err = xorm.NewEngine("sqlite3", ".mock.db")
//		if err != nil {
//			logger.Error(err)
//			panic(err)
//		}
//	})
//
//	model := reflect.New(scope.GetModelStruct().ModelType).Interface()
//	//m.RegisterModels(model)
//	m.dumper.Sync2(model)
//	m.dbRecorder.AutoMigrate(model)
//	rValue := reflect.ValueOf(scope.Value)
//	if rValue.Kind() == reflect.Ptr {
//		rValue = rValue.Elem()
//	}
//	if rValue.Kind() == reflect.Slice || rValue.Kind() == reflect.Array {
//		for i := 0; i < rValue.Len(); i++ {
//			m.dbRecorder.Create(rValue.Index(i).Interface())
//		}
//		return
//	}
//	if rValue.Kind() == reflect.Struct {
//		m.dbRecorder.Create(rValue.Interface())
//	}
//}

func (m *MockGORM) SaveRecordToFile(dir string) {
	m.util.SaveRecordToFile(dir, m.util.DumpFromRecordInfo(m.recorderSQLDB, m.DumpRecorderInfo()))
}

func (m *MockGORM) DoRecord(scope *gorm.Scope) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("%v", err))
		}
	}()

	//m.onceRecorder.Do(func() {
	//	var err error
	//	if m.dbType == "mysql" {
	//		t, dsn := m.GetDSN()
	//		m.dbRecorder, err = gorm.Open(t, dsn)
	//	} else {
	//		m.dbRecorder, err = gorm.Open("sqlite3", ".mock.db")
	//	}
	//	m.dbRecorder.SingularTable(true)
	//	m.dumper, err = xorm.NewEngine("sqlite3", ".mock.db")
	//	if err != nil {
	//		logger.Error(err)
	//		panic(err)
	//	}
	//})

	if m.recorderSQLDB == nil {
		m.recorderSQLDB = scope.DB().DB()
	}

	m.recordLock.Lock()
	defer m.recordLock.Unlock()
	tableName := scope.TableName()
	if tableName == "" {
		return
	}
	if _, ok := m.recorder[tableName]; !ok {
		m.recorder[tableName] = mapset.NewSet()
	}

	//model := reflect.New(scope.GetModelStruct().ModelType).Interface()
	////m.RegisterModels(model)
	//m.dumper.Sync2(model)
	//m.dbRecorder.AutoMigrate(model)
	rValue := reflect.ValueOf(scope.Value)
	if !rValue.IsValid() {
		return
	}
	if rValue.Kind() == reflect.Ptr {
		rValue = rValue.Elem()
		if !rValue.IsValid() {
			return
		}
	}
	id := ""
	if rValue.Kind() == reflect.Slice || rValue.Kind() == reflect.Array {
		if rValue.Len() == 0 {
			return
		}
		item := rValue.Index(0)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		if item.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < item.NumField(); i++ {
			id = item.Type().Field(i).Name
			if id == "id" || id == "ID" || id == "Id" {
				break
			}
		}
		if id == "" {
			return
		}
		for i := 0; i < rValue.Len(); i++ {
			//m.dbRecorder.Create(rValue.Index(i).Interface())
			item := rValue.Index(i)
			if item.IsValid() && item.Kind() == reflect.Ptr {
				item = item.Elem()
			}
			if item.IsValid() && item.FieldByName(id).Kind() == reflect.Ptr {
				m.recorder[tableName].Add(item.FieldByName(id).Elem().Interface())
			} else {
				m.recorder[tableName].Add(item.FieldByName(id).Interface())
			}

		}
		return
	}
	if rValue.Kind() == reflect.Struct {
		for i := 0; i < rValue.NumField(); i++ {
			id = rValue.Type().Field(i).Name
			if id == "id" || id == "ID" || id == "Id" {
				break
			}
		}
		if id == "" {
			return
		}
		if !rValue.FieldByName(id).IsValid() {
			return
		}
		if rValue.FieldByName(id).Kind() == reflect.Ptr {
			m.recorder[tableName].Add(rValue.FieldByName(id).Elem().Interface())
		} else {
			m.recorder[tableName].Add(rValue.FieldByName(id).Interface())
		}

		//m.dbRecorder.Create(rValue.Interface())
	}
	//scope.HasColumn("id") || scope
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
				logger.Panic(fmt.Sprintf("model should be struct prt"))
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
				logger.Error(sql)
				panic(err)
			}
		}
	}
	for _, filePath := range getFilesBySuffix(m.pathToSqlFileName, "sql") {
		sqlText := m.readMockSQl(filePath)
		sqls := m.parseMockSQL(sqlText)
		for _, sql := range sqls {
			if strings.TrimSpace(sql) == "" {
				continue
			}
			err := m.db.Exec(sql).Error
			if err != nil {
				logger.Error(filePath)
				logger.Error(sql)
				panic(err)
			}
		}
		logger.Log(fmt.Sprintf("sql file %v is loaded", filePath))
	}

}

// ReadMockSQl read sql file to string
func (m *MockGORM) readMockSQl(filePath string) string {
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

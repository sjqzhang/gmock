package mockdb

import (
	"database/sql"
	"fmt"
	mapset "github.com/deckarep/golang-set"
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
	"sync"
	"time"
)

type MockGORMV2 struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *gorm.DB
	dbRecorder        *gorm.DB
	onceRecorder      sync.Once
	dbType            string
	dsn               string
	models            []interface{}
	util              *util.DBUtil
	resetHandler      func(resetHandler *MockGORMV2)
	schema            string
	//dumper            *xorm.Engine
	recorder   map[string]mapset.Set
	recordLock sync.Mutex
}

func NewMockGORMV2(pathToSqlFileName string, resetHandler func(orm *MockGORMV2)) *MockGORMV2 {
	mock := MockGORMV2{
		pathToSqlFileName: pathToSqlFileName,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
		recorder:          make(map[string]mapset.Set),
		recordLock:        sync.Mutex{},
		//onceRecorder:      sync.Once{},
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
			mock.dsn = fmt.Sprintf("root:root@tcp(127.0.0.1:%v)/mock?charset=utf8&parseTime=True&loc=Local", i)
			mock.dbType = "mysql"
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

func (m *MockGORMV2) SaveRecordToFile(db *sql.DB, dir string) {
	m.util.SaveRecordToFile(dir, m.util.DumpFromRecordInfo(db, m.DumpRecorderInfo()))
}

func (m *MockGORMV2) DumpRecorderInfo() map[string][]string {
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

func (m *MockGORMV2) DoRecord(scope *gorm.DB) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(fmt.Sprintf("%v", err))
		}
	}()

	m.recordLock.Lock()
	defer m.recordLock.Unlock()
	tableName := scope.Statement.Table
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
	rValue := reflect.ValueOf(scope.Statement.Model)
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

		err := m.db.Debug().AutoMigrate(model)
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
				logger.Error(sqlStr)
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

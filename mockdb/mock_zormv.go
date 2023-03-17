package mockdb

import (
	"context"
	"database/sql"
	"fmt"
	"gitee.com/chunanyong/zorm"
	"github.com/sjqzhang/gmock/util"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unsafe"
)

var ctxKey string = "contextDBConnectionValueKey"

type MockZORM struct {
	pathToSqlFileName string `json:"path_to_sql_file_name"`
	db                *zorm.DBDao
	dbType            string
	dsn               string
	models            []interface{}
	util              *util.DBUtil
	resetHandler      func(resetHandler *MockZORM)
	schema            string
	ctx               context.Context
}

func NewMockZORM(pathToSqlFileName string, resetHandler func(orm *MockZORM)) *MockZORM {
	mock := MockZORM{
		pathToSqlFileName: pathToSqlFileName,
		models:            make([]interface{}, 0),
		resetHandler:      resetHandler,
	}
	var err error
	var db *zorm.DBDao
	//ns := schema.NamingStrategy{
	//	SingularTable: true,
	//}
	conf := &zorm.DataSourceConfig{}
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
			conf.DBType = mock.dbType
			conf.DSN = mock.dsn
			conf.DriverName = "mysql"
			db, err = zorm.NewDBDao(conf)
			//db, err = gorm.Open(mysql.Open(mock.dsn), &gorm.Config{NamingStrategy: ns})
			break
		}

	} else {
		mock.dbType = "sqlite3"
		mock.dsn = "file::memory:?cache=shared"
		conf.DBType = mock.dbType
		conf.DSN = mock.dsn
		conf.DriverName = mock.dbType
		db, err = zorm.NewDBDao(conf)
		//db, err = gorm.Open(sqlite.Open(mock.dsn), &gorm.Config{
		//	NamingStrategy: ns,
		//})
	}
	mock.ctx = context.Background()
	mock.ctx = context.WithValue(mock.ctx, ctxKey, db)
	if err != nil {
		panic(err)
	}
	mock.db = db
	return &mock
}

//func renew2() *gorm.DB {
//	var err error
//	ns := schema.NamingStrategy{
//		SingularTable: true,
//	}
//	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
//		NamingStrategy: ns,
//	})
//	if err != nil {
//		panic(err)
//	}
//	return db
//}

// ResetAndInit 初始化数据库及表数据
func (m *MockZORM) ResetAndInit() {
	//m.db = renew2()

	m.dropTables()
	m.initModels()
	m.initSQL()
	if m.resetHandler != nil {
		m.resetHandler(m)
	}
}

// GetGormDB 获取Gorm实例
func (m *MockZORM) GetDBDao() *zorm.DBDao {
	return m.db
}
func (m *MockZORM) dropTables() {
	for _, model := range m.models {
		//m.db.Migrator().DropTable(model)
		_ = model
	}
}
func (m *MockZORM) GetDSN() (dbType string, dsn string) {
	dbType = m.dbType
	dsn = m.dsn
	return
}

func (m *MockZORM) GetDBUtil() *util.DBUtil {
	return m.util
}

func (m *MockZORM) InitSchemas(sqlSchema string) {
	if util.Util.IsExist(sqlSchema) {
		data,err:= util.Util.ReadBinFile(sqlSchema)
		if err != nil {
			panic(err)
		}
		sqlSchema = string(data)
	}
	m.schema = sqlSchema
}

// GetSqlDB  获取*sql.DB实例
func (m *MockZORM) GetSqlDB() *sql.DB {
	rs:=reflect.ValueOf( m.ctx.Value(ctxKey))
	rs2 := reflect.New(rs.Type()).Elem()
	rs2.Set(rs)
	rf:= rs2.Elem().FieldByName("dataSource")
	v := reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem().Elem().FieldByName("DB")
	return v.Interface().(*sql.DB)

}

// RegisterModels 注册模型
func (m *MockZORM) RegisterModels(models ...interface{}) {
	panic("not implemented,please use InitSchemas with sql content")
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
func (m *MockZORM) initModels() {
	if m.db == nil {
		panic("warning: call ResetAndInit func first!!!!!")
	}
	for _, model := range m.models {
		// TODO
		_ = model
		//err := m.db.AutoMigrate(model)
		//if err != nil {
		//	panic(err)
		//}
	}
}
func (m *MockZORM) initSQL() {
	db:=m.GetSqlDB()
	if m.schema != "" {
		sqls := m.parseMockSQL(m.schema)
		for _, sql := range sqls {
			_,err := db.Exec(sql)
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
			_,err :=db.Exec(sqlStr)
			if err != nil {
				logger.Error(filePath)
				panic(err)
			}
		}
		logger.Error(fmt.Sprintf("sql file %v is loaded", filePath))
	}
}

// ReadMockSQl read sql file to string
func (m *MockZORM) readMockSQl(filePath string) string {
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
func (m *MockZORM) parseMockSQL(sqlText string) []string {
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

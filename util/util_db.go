package util

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/auth"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	sqlm "github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/information_schema"
	"github.com/jinzhu/gorm"
	gormv2 "gorm.io/gorm"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type DBUtil struct {
}

func NewDBUtil() *DBUtil {
	return &DBUtil{}
}

var regSelect = regexp.MustCompile(`(?i)^select`)

func (u *DBUtil) SelectToInsertSQLV1(scope *gorm.Scope) (string, []string) {
	var sqls []string
	rValue := reflect.ValueOf(scope.Value)
	rType := scope.GetModelStruct().ModelType
	sql := scope.SQL
	tableName := scope.TableName()
	if regSelect.MatchString(sql) {
		return u.selectToInsertSQL(rValue, rType, tableName)
	}
	return tableName, sqls
}

func (u *DBUtil) selectToInsertSQL(rValue reflect.Value, rType reflect.Type, tableName string) (string, []string) {
	var sqls []string
	if rValue.Kind() != reflect.Ptr {
		return tableName, sqls
	}

	if rValue.Elem().Kind() == reflect.Slice || rValue.Elem().Kind() == reflect.Array {
		for i := 0; i < rValue.Elem().Len(); i++ {
			obj := rValue.Elem().Index(i)
			if obj.Kind() == reflect.Struct {
				var fields []string
				var fieldValues []string
				for j := 0; j < obj.NumField(); j++ {
					name := rType.Field(j).Tag.Get("json")
					fields = append(fields, fmt.Sprintf("`%v`", name))
					field := obj.Field(j)
					switch field.Kind() {

					case reflect.String:
						fieldValues = append(fieldValues, fmt.Sprintf("'%v'", field.Interface()))
					case reflect.Int64, reflect.Int32, reflect.Int, reflect.Float32, reflect.Float64, reflect.Bool:
						fieldValues = append(fieldValues, fmt.Sprintf("%v", field.Interface()))
					default:
						fieldValues = append(fieldValues, fmt.Sprintf("'%v'", field.Interface()))

					}

				}
				sqls = append(sqls, fmt.Sprintf("INSERT INTO  `%v`(%v) VALUES(%v)", tableName, strings.Join(fields, ","), strings.Join(fieldValues, ",")))
			}
		}
		return tableName, sqls
	}
	if rValue.Elem().Kind() == reflect.Struct { //对struct处理
		var fields []string
		var fieldValues []string
		for i := 0; i < rType.NumField(); i++ {
			name := rType.Field(i).Tag.Get("json")
			fields = append(fields, fmt.Sprintf("`%v`", name))
			field := rValue.Elem().Field(i)
			switch field.Kind() {

			case reflect.String:
				fieldValues = append(fieldValues, fmt.Sprintf("'%v'", field.Interface()))
			case reflect.Int64, reflect.Int32, reflect.Int, reflect.Float32, reflect.Float64, reflect.Bool:
				fieldValues = append(fieldValues, fmt.Sprintf("%v", field.Interface()))
			default:
				fieldValues = append(fieldValues, fmt.Sprintf("'%v'", field.Interface()))

			}

		}
		sqls = append(sqls, fmt.Sprintf("INSERT INTO `%v`(%v) VALUES(%v)", tableName, strings.Join(fields, ","), strings.Join(fieldValues, ",")))
	}
	return tableName, sqls
}

func (u *DBUtil) SelectToInsertSQLV2(db *gormv2.DB) (string, []string) {
	var sqls []string
	rType := db.Statement.Schema.ModelType
	rValue := reflect.ValueOf(db.Statement.Model)
	if regSelect.MatchString(db.Statement.SQL.String()) {
		return u.selectToInsertSQL(rValue, rType, db.Statement.Table)
	}
	return db.Statement.Table, sqls
}

func (u *DBUtil) QueryListBySQL(db *sql.DB, sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cys, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for rows.Next() {
		l := len(cys)
		vals := make([]interface{}, l)
		valPtr := make([]interface{}, l)
		for i, _ := range vals {
			valPtr[i] = &vals[i]
		}
		row := make(map[string]interface{}, l)
		rows.Scan(valPtr...)
		for i, c := range cys {
			row[c.Name()] = vals[i]
		}
		result = append(result, row)
	}
	return result, nil
}

func (u *DBUtil) QueryOneBySQL(db *sql.DB, sqlStr string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cys, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	l := len(cys)
	vals := make([]interface{}, l)
	row := make(map[string]interface{}, l)
	for rows.Next() {

		valPtr := make([]interface{}, l)
		for i, _ := range vals {
			valPtr[i] = &vals[i]
		}

		rows.Scan(valPtr...)
		for i, c := range cys {
			row[c.Name()] = vals[i]
		}
		break
	}
	return row, nil
}

func (u *DBUtil) QueryObjectBySQL(db *sql.DB, obj interface{}, sqlStr string, args ...interface{}) error {
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		errors.New("obj must be a pointer")
	}
	elem := reflect.ValueOf(obj).Elem()
	if elem.Kind() == reflect.Slice || elem.Kind() == reflect.Array {
		rows, err := u.QueryListBySQL(db, sqlStr, args...)
		if err != nil {
			return err
		}
		data, err := json.Marshal(rows)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, obj)
		if err != nil {
			return err
		}
	} else if elem.Kind() == reflect.Struct {
		row, err := u.QueryOneBySQL(db, sqlStr, args...)
		if err != nil {
			return err
		}
		data, err := json.Marshal(row)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, obj)
		if err != nil {
			return err
		}
	} else {
		errors.New("obj must be a pointer to struct or slice")
	}
	return nil
}

func (u *DBUtil) ExecSQL(db *sql.DB, sqlStr string, args ...interface{}) (sql.Result, error) {
	return db.Exec(sqlStr, args...)
}

func (m *DBUtil) ParseSQLText(sqlText string) []string {
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

func (u *DBUtil) ReadFile(filePath string) string {
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

func (u *DBUtil) IntToPr(i int) *int {
	i2 := i
	return &i2
}

func (u *DBUtil) StrToPr(s string) *string {
	s2 := s
	return &s2
}

func (u *DBUtil) RunMySQLServer(dbName string, dbPort int, block bool) {
	engine := sqle.NewDefault(
		sqlm.NewDatabaseProvider(
			memory.NewDatabase(dbName),
			information_schema.NewInformationSchemaDatabase(),
		))
	config := server.Config{
		Protocol: "tcp",
		Address:  fmt.Sprintf("0.0.0.0:%v", dbPort),
		Auth:     &auth.None{},
	}
	s, err := server.NewDefaultServer(config, engine)
	if err != nil {
		panic(err)
	}
	if block {
		s.Start()
	} else {
		go s.Start()
	}
}

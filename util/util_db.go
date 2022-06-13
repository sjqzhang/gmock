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
	"io"
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

			if (obj.Kind() == reflect.Ptr && obj.Elem().Kind() == reflect.Struct) || obj.Kind() == reflect.Struct {
				sqls = append(sqls, u.getStructSQL(rType, obj, tableName))
			}

		}
		return tableName, sqls
	}

	if rValue.Elem().Kind() == reflect.Struct {
		sqls = append(sqls, u.getStructSQL(rType, rValue, tableName))
	}

	return tableName, sqls
}

func (u *DBUtil) getTagAttr(f reflect.StructField, tagName string, tagAttr string) (string, bool) {
	if tag, ok := f.Tag.Lookup(tagName); ok {
		m := make(map[string]string)
		tags := strings.Split(tag, ";")
		for _, t := range tags {
			kvs := strings.Split(t, ":")
			if len(kvs) == 1 {
				m[kvs[0]] = ""
			}
			if len(kvs) == 2 {
				m[kvs[0]] = kvs[1]
			}
		}
		v, o := m[tagAttr]
		return v, o
	}
	return "", false
}

func (u *DBUtil) Dump(db *sql.DB, tables []string, w io.Writer) {
	var sqls []string
	for _, table := range tables {
		rows, err := u.QueryListBySQL(db, fmt.Sprintf("select * from %v", table))
		if err != nil {
			log.Println(err)
			continue
		}
		for _, row := range rows {
			var fieldNames []string
			var fieldValues []string
			for name, value := range row {
				fieldNames = append(fieldNames, fmt.Sprintf("`%v`", name))
				v := ""
				switch value.(type) {
				case int, int64, float64, float32, bool:
					v = fmt.Sprintf("%v", value)
				case string:
					v = fmt.Sprintf("%v", strings.Replace(value.(string), "'", "\\'", -1))
				default:
					v = fmt.Sprintf("%v", value)
					v = fmt.Sprintf("%v", strings.Replace(v, "'", "\\'", -1))
				}
				fieldValues = append(fieldValues, v)
			}
			sql := fmt.Sprintf("INSERT INTO `%v` (%v) VALUES (%v);\n", table, strings.Join(fieldNames, ","), strings.Join(fieldValues, ","))
			sqls = append(sqls, sql)
		}
	}

	w.Write([]byte(strings.Join(sqls, "")))

}

func (u *DBUtil) getStructSQL(rType reflect.Type, rValue reflect.Value, tableName string) string {
	if rValue.Kind() == reflect.Ptr {
		rValue = rValue.Elem()
	}
	if rValue.Kind() == reflect.Struct { //对struct处理
		var fields []string
		var fieldValues []string
		var name string
		var field reflect.Value
		var ok bool
		for i := 0; i < rType.NumField(); i++ {
			name = rType.Field(i).Tag.Get("json")
			field = rValue.Field(i)
			if rType.Field(i).Anonymous {
				fmt.Println(rValue.Field(i).Kind().String())
				if rValue.Field(i).Kind() == reflect.Struct {
					for j := 0; j < rValue.Field(i).Type().NumField(); j++ {
						//name = rValue.Field(i).Type().Field(j).Tag.Get("gorm")
						name, ok = u.getTagAttr(rValue.Field(i).Type().Field(j), "gorm", "column")
						if name == "" || !ok {
							//name, ok = u.getTagAttr(rValue.Field(i).Type().Field(j), "gorm", "column")
							//if !ok {
							//	continue
							//}
							//strings.Split(rValue.Field(i).Type().Field(j).Tag.Get("gorm"), ";")
							name = rValue.Field(i).Type().Field(j).Tag.Get("json")
							if name=="" {
								continue
							}
						}
						if rValue.Field(i).Type().Kind() == reflect.Ptr {
							field = rValue.Field(i).Elem().Field(j)
						} else {
							field = rValue.Field(i).Field(j)
						}
						if name == "" {
							continue
						}
						fields = append(fields, fmt.Sprintf("`%v`", name))
						switch field.Kind() {

						case reflect.String:
							fieldValues = append(fieldValues, fmt.Sprintf("'%v'", strings.Replace(fmt.Sprintf("%v", field.Interface()), "'", "\\'", -1)))
						case reflect.Int64, reflect.Int32, reflect.Int, reflect.Float32, reflect.Float64, reflect.Bool:
							fieldValues = append(fieldValues, fmt.Sprintf("%v", field.Interface()))
						default:
							fieldValues = append(fieldValues, fmt.Sprintf("'%v'", strings.Replace(fmt.Sprintf("%v", field.Interface()), "'", "\\'", -1)))

						}
					}
				}
			}

			if name == "" {
				continue
			}
			fields = append(fields, fmt.Sprintf("`%v`", name))
			switch field.Kind() {

			case reflect.String:
				fieldValues = append(fieldValues, fmt.Sprintf("'%v'", strings.Replace(fmt.Sprintf("%v", field.Interface()), "'", "\\'", -1)))
			case reflect.Int64, reflect.Int32, reflect.Int, reflect.Float32, reflect.Float64, reflect.Bool:
				fieldValues = append(fieldValues, fmt.Sprintf("%v", field.Interface()))
			default:
				fieldValues = append(fieldValues, fmt.Sprintf("'%v'", strings.Replace(fmt.Sprintf("%v", field.Interface()), "'", "\\'", -1)))

			}

		}
		return fmt.Sprintf("INSERT INTO `%v`(%v) VALUES(%v)", tableName, strings.Join(fields, ","), strings.Join(fieldValues, ","))
	}
	return ""
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

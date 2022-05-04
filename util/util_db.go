package util

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
)

type DBUtil struct {
}

func NewDBUtil() *DBUtil {
	return &DBUtil{}
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

func (u *DBUtil) QueryObjectBySQL(db *sql.DB, obj interface, sqlStr string, args ...interface{}) error {
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

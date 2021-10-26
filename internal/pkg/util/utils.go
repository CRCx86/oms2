package util

import (
	"encoding/json"
	"github.com/jackc/pgx/v4"
)

func ParseRowQuery(rows pgx.Rows) ([]map[string]interface{}, error) {
	fields := rows.FieldDescriptions()
	var columns []string
	for _, field := range fields {
		columns = append(columns, string(field.Name))
	}

	count := len(columns)
	values := make([]interface{}, count)
	valuesPointers := make([]interface{}, count)

	var objects []map[string]interface{}
	for rows.Next() {
		for i := range columns {
			valuesPointers[i] = &values[i]
		}

		err := rows.Scan(valuesPointers...)
		if err != nil {
			return nil, err
		}

		object := map[string]interface{}{}
		for i, column := range columns {
			val := values[i]
			object[column] = val
		}

		objects = append(objects, object)
	}
	return objects, nil
}

func ToEntity(data []map[string]interface{}, entity interface{}) error {

	bytes, err := json.Marshal(data[0])
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, &entity)
	if err != nil {
		return err
	}
	return nil
}

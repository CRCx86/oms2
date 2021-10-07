package util

import "github.com/jackc/pgx/v4"

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
	object := map[string]interface{}{}
	for rows.Next() {
		for i := range columns {
			valuesPointers[i] = &values[i]
		}

		err := rows.Scan(valuesPointers...)
		if err != nil {
			return nil, err
		}

		for i, column := range columns {
			val := values[i]
			object[column] = val
		}

		objects = append(objects, object)
	}
	return objects, nil
}

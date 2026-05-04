package export

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/XingfenD/tag_file/pkg/storage"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ExportToSQL(w io.Writer) error {
	db := storage.GetDB()
	rawDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	if _, err := io.WriteString(w, "BEGIN TRANSACTION;\n\n"); err != nil {
		return err
	}

	tables, err := getTableNames(rawDB)
	if err != nil {
		return fmt.Errorf("读取表列表失败: %w", err)
	}

	for _, name := range tables {
		if err := dumpCreateTable(w, rawDB, name); err != nil {
			return err
		}
		if err := dumpTableData(w, rawDB, name); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(w, "COMMIT;\n"); err != nil {
		return err
	}
	return nil
}

func getTableNames(rdb *sql.DB) ([]string, error) {
	rows, err := rdb.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func dumpCreateTable(w io.Writer, rdb *sql.DB, table string) error {
	var sqlStr sql.NullString
	err := rdb.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name = ?", table).Scan(&sqlStr)
	if err != nil {
		return fmt.Errorf("读取表结构失败 %s: %w", table, err)
	}
	if sqlStr.Valid {
		_, err := io.WriteString(w, sqlStr.String+";\n\n")
		return err
	}
	return nil
}

func dumpTableData(w io.Writer, rdb *sql.DB, table string) error {
	rows, err := rdb.Query("SELECT * FROM " + table)
	if err != nil {
		return fmt.Errorf("查询表 %s 失败: %w", table, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	colList := strings.Join(quoteCols(cols), ", ")

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

		line := formatInsert(table, colList, vals)
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return rows.Err()
}

func formatInsert(table, colList string, vals []interface{}) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = formatValue(v)
	}
	return fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s);\n", table, colList, strings.Join(parts, ", "))
}

func formatValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return escapeString(string(t))
	case string:
		return escapeString(t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float64:
		return fmt.Sprintf("%g", t)
	default:
		return fmt.Sprintf("'%v'", t)
	}
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	return "'" + s + "'"
}

func quoteCols(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = fmt.Sprintf("%q", c)
	}
	return out
}

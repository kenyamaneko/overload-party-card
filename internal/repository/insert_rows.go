package repository

import (
	"context"
	"fmt"
	"strings"
)

// insertRows は 1 回の INSERT 文で複数行を挿入します。行が無いときは何もしません。
// 各行の値は columns と同じ順に並べます。conflictClause は文末に付ける ON CONFLICT 句で、不要なときは空文字を渡します。
func insertRows(ctx context.Context, db dbtx, table string, columns []string, rows [][]any, conflictClause string) error {
	if len(rows) == 0 {
		return nil
	}

	var statement strings.Builder
	fmt.Fprintf(&statement, "INSERT INTO %s (%s) VALUES ", table, strings.Join(columns, ", "))

	args := make([]any, 0, len(rows)*len(columns))
	for i, row := range rows {
		if i > 0 {
			statement.WriteString(",")
		}
		statement.WriteString("(")
		for j := range columns {
			if j > 0 {
				statement.WriteString(",")
			}
			fmt.Fprintf(&statement, "$%d", len(args)+1)
			args = append(args, row[j])
		}
		statement.WriteString(")")
	}
	if conflictClause != "" {
		statement.WriteString(" ")
		statement.WriteString(conflictClause)
	}

	_, err := db.Exec(ctx, statement.String(), args...)
	return err
}

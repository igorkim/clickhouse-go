package std

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStdDate32(t *testing.T) {
	if conn, err := sql.Open("clickhouse", "clickhouse://127.0.0.1:9000"); assert.NoError(t, err) {
		if err := checkMinServerVersion(conn, 21, 9); err != nil {
			t.Skip(err.Error())
			return
		}
		const ddl = `
			CREATE TABLE test_date32 (
				  ID   UInt8
				, Col1 Date32
				, Col2 Nullable(Date32)
				, Col3 Array(Date32)
				, Col4 Array(Nullable(Date32))
			) Engine Memory
		`
		type result struct {
			ColID uint8 `ch:"ID"`
			Col1  time.Time
			Col2  *time.Time
			Col3  []time.Time
			Col4  []*time.Time
		}
		if _, err := conn.Exec("DROP TABLE IF EXISTS test_date32"); assert.NoError(t, err) {
			if _, err := conn.Exec(ddl); assert.NoError(t, err) {
				scope, err := conn.Begin()
				if !assert.NoError(t, err) {
					return
				}
				if batch, err := scope.Prepare("INSERT INTO test_date32"); assert.NoError(t, err) {
					var (
						date1, _ = time.Parse("2006-01-02 15:04:05", "2283-11-11 00:00:00")
						date2, _ = time.Parse("2006-01-02 15:04:05", "1925-01-01 00:00:00")
					)
					if _, err := batch.Exec(uint8(1), date1, &date2, []time.Time{date2}, []*time.Time{&date2, nil, &date1}); !assert.NoError(t, err) {
						return
					}
					if _, err := batch.Exec(uint8(2), date2, nil, []time.Time{date1}, []*time.Time{nil, nil, &date2}); !assert.NoError(t, err) {
						return
					}
					if err := scope.Commit(); assert.NoError(t, err) {
						var (
							result1 result
							result2 result
						)
						if err := conn.QueryRow("SELECT * FROM test_date32 WHERE ID = $1", 1).Scan(
							&result1.ColID,
							&result1.Col1,
							&result1.Col2,
							&result1.Col3,
							&result1.Col4,
						); assert.NoError(t, err) {
							if assert.Equal(t, date1, result1.Col1) {
								assert.Equal(t, "UTC", result1.Col1.Location().String())
								assert.Equal(t, date2, *result1.Col2)
								assert.Equal(t, []time.Time{date2}, result1.Col3)
								assert.Equal(t, []*time.Time{&date2, nil, &date1}, result1.Col4)
							}
						}
						if err := conn.QueryRow("SELECT * FROM test_date32 WHERE ID = $1", 2).Scan(
							&result2.ColID,
							&result2.Col1,
							&result2.Col2,
							&result2.Col3,
							&result2.Col4,
						); assert.NoError(t, err) {
							if assert.Equal(t, date2, result2.Col1) {
								assert.Equal(t, "UTC", result2.Col1.Location().String())
								if assert.Nil(t, result2.Col2) {
									assert.Equal(t, []time.Time{date1}, result2.Col3)
									assert.Equal(t, []*time.Time{nil, nil, &date2}, result2.Col4)
								}
							}
						}
					}
				}
			}
		}
	}
}

package std

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdEnum(t *testing.T) {
	if conn, err := sql.Open("clickhouse", "clickhouse://127.0.0.1:9000"); assert.NoError(t, err) {
		const ddl = `
			CREATE TABLE test_enum (
				  Col1 Enum  ('hello'   = 1,  'world' = 2)
				, Col2 Enum8 ('click'   = 5,  'house' = 25)
				, Col3 Enum16('house' = 10,   'value' = 50)
				, Col4 Array(Enum8  ('click' = 1, 'house' = 2))
				, Col5 Array(Enum16 ('click' = 1, 'house' = 2))
				, Col6 Array(Nullable(Enum8  ('click' = 1, 'house' = 2)))
				, Col7 Array(Nullable(Enum16 ('click' = 1, 'house' = 2)))
			) Engine Memory
		`
		if _, err := conn.Exec("DROP TABLE IF EXISTS test_enum"); assert.NoError(t, err) {
			if _, err := conn.Exec(ddl); assert.NoError(t, err) {
				scope, err := conn.Begin()
				if !assert.NoError(t, err) {
					return
				}
				if batch, err := scope.Prepare("INSERT INTO test_enum"); assert.NoError(t, err) {
					var (
						col1Data = "hello"
						col2Data = "click"
						col3Data = "house"
						col4Data = []string{"click", "house"}
						col5Data = []string{"house", "click"}
						col6Data = []*string{&col2Data, nil, &col3Data}
						col7Data = []*string{&col3Data, nil, &col2Data}
					)
					if _, err := batch.Exec(
						col1Data,
						col2Data,
						col3Data,
						col4Data,
						col5Data,
						col6Data,
						col7Data,
					); assert.NoError(t, err) {
						if err := scope.Commit(); assert.NoError(t, err) {
							var (
								col1 string
								col2 string
								col3 string
								col4 []string
								col5 []string
								col6 []*string
								col7 []*string
							)
							if err := conn.QueryRow("SELECT * FROM test_enum").Scan(
								&col1, &col2, &col3, &col4,
								&col5, &col6, &col7,
							); assert.NoError(t, err) {
								assert.Equal(t, col1Data, col1)
								assert.Equal(t, col2Data, col2)
								assert.Equal(t, col3Data, col3)
								assert.Equal(t, col4Data, col4)
								assert.Equal(t, col5Data, col5)
								assert.Equal(t, col6Data, col6)
								assert.Equal(t, col7Data, col7)
							}
						}
					}
				}
			}
		}
	}
}

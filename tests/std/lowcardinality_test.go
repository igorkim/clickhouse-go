package std

import (
	"context"
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestStdLowCardinality(t *testing.T) {
	ctx := clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
		"allow_suspicious_low_cardinality_types": 1,
	}))
	if conn, err := sql.Open("clickhouse", "clickhouse://127.0.0.1:9000"); assert.NoError(t, err) {
		if err := checkMinServerVersion(conn, 19, 11); err != nil {
			t.Skip(err.Error())
			return
		}
		const ddl = `
		CREATE TABLE test_lowcardinality (
			  Col1 LowCardinality(String)
			, Col2 LowCardinality(FixedString(2))
			, Col3 LowCardinality(DateTime)
			, Col4 LowCardinality(Int32)
			, Col5 Array(LowCardinality(String))
			, Col6 Array(Array(LowCardinality(String)))
			, Col7 LowCardinality(Nullable(String))
			, Col8 Array(Array(LowCardinality(Nullable(String))))
		) Engine Memory
		`
		if _, err := conn.Exec("DROP TABLE IF EXISTS test_lowcardinality"); assert.NoError(t, err) {
			if _, err := conn.ExecContext(ctx, ddl); assert.NoError(t, err) {
				scope, err := conn.Begin()
				if !assert.NoError(t, err) {
					return
				}
				if batch, err := scope.Prepare("INSERT INTO test_lowcardinality"); assert.NoError(t, err) {
					var (
						rnd       = rand.Int31()
						timestamp = time.Now()
					)
					for i := 0; i < 10; i++ {
						var (
							col1Data = timestamp.String()
							col2Data = "RU"
							col3Data = timestamp.Add(time.Duration(i) * time.Minute)
							col4Data = rnd + int32(i)
							col5Data = []string{"A", "B", "C"}
							col6Data = [][]string{
								[]string{"Q", "W", "E"},
								[]string{"R", "T", "Y"},
							}
							col7Data = &col2Data
							col8Data = [][]*string{
								[]*string{&col2Data, nil, &col2Data},
								[]*string{nil, &col2Data, nil},
							}
						)
						if i%2 == 0 {
							if _, err := batch.Exec(col1Data, col2Data, col3Data, col4Data, col5Data, col6Data, col7Data, col8Data); !assert.NoError(t, err) {
								return
							}
						} else {
							if _, err := batch.Exec(col1Data, col2Data, col3Data, col4Data, col5Data, col6Data, nil, col8Data); !assert.NoError(t, err) {
								return
							}
						}
					}
					if assert.NoError(t, scope.Commit()) {
						var count uint64
						if err := conn.QueryRow("SELECT COUNT() FROM test_lowcardinality").Scan(&count); assert.NoError(t, err) {
							assert.Equal(t, uint64(10), count)
						}
						for i := 0; i < 10; i++ {
							var (
								col1 string
								col2 string
								col3 time.Time
								col4 int32
								col5 []string
								col6 [][]string
								col7 *string
								col8 [][]*string
							)
							if err := conn.QueryRow("SELECT * FROM test_lowcardinality WHERE Col4 = $1", rnd+int32(i)).Scan(&col1, &col2, &col3, &col4, &col5, &col6, &col7, &col8); assert.NoError(t, err) {
								assert.Equal(t, timestamp.String(), col1)
								assert.Equal(t, "RU", col2)
								assert.Equal(t, timestamp.Add(time.Duration(i)*time.Minute).Truncate(time.Second), col3)
								assert.Equal(t, rnd+int32(i), col4)
								assert.Equal(t, []string{"A", "B", "C"}, col5)
								assert.Equal(t, [][]string{
									[]string{"Q", "W", "E"},
									[]string{"R", "T", "Y"},
								}, col6)
								switch {
								case i%2 == 0:
									assert.Equal(t, &col2, col7)
								default:
									assert.Nil(t, col7)
								}
								col2Data := "RU"
								assert.Equal(t, [][]*string{
									[]*string{&col2Data, nil, &col2Data},
									[]*string{nil, &col2Data, nil},
								}, col8)
							}
						}
					}
				}
			}
		}
	}
}

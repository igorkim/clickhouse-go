package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			Compression: &clickhouse.Compression{
				Method: clickhouse.CompressionLZ4,
			},
			//Debug: true,
		})
	)
	if assert.NoError(t, err) {
		if err := checkMinServerVersion(conn, 21, 9); err != nil {
			t.Skip(err.Error())
			return
		}
		const ddl = `
		CREATE TABLE test_map (
			  Col1 Map(String, UInt64)
			, Col2 Map(String, UInt64)
			, Col3 Map(String, UInt64)
			, Col4 Array(Map(String, String))
			, Col5 Map(LowCardinality(String), LowCardinality(UInt64))
		) Engine Memory
		`
		if err := conn.Exec(ctx, "DROP TABLE IF EXISTS test_map"); assert.NoError(t, err) {
			if err := conn.Exec(ctx, ddl); assert.NoError(t, err) {
				if batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_map"); assert.NoError(t, err) {
					var (
						col1Data = map[string]uint64{
							"key_col_1_1": 1,
							"key_col_1_2": 2,
						}
						col2Data = map[string]uint64{
							"key_col_2_1": 10,
							"key_col_2_2": 20,
						}
						col3Data = map[string]uint64{}
						col4Data = []map[string]string{
							map[string]string{"A": "B"},
							map[string]string{"C": "D"},
						}
						col5Data = map[string]uint64{
							"key_col_5_1": 100,
							"key_col_5_2": 200,
						}
					)
					if err := batch.Append(col1Data, col2Data, col3Data, col4Data, col5Data); assert.NoError(t, err) {
						if assert.NoError(t, batch.Send()) {
							var (
								col1 map[string]uint64
								col2 map[string]uint64
								col3 map[string]uint64
								col4 []map[string]string
								col5 map[string]uint64
							)
							if err := conn.QueryRow(ctx, "SELECT * FROM test_map").Scan(&col1, &col2, &col3, &col4, &col5); assert.NoError(t, err) {
								assert.Equal(t, col1Data, col1)
								assert.Equal(t, col2Data, col2)
								assert.Equal(t, col3Data, col3)
								assert.Equal(t, col4Data, col4)
								assert.Equal(t, col5Data, col5)
							}
						}
					}
				}
			}
		}
	}
}
func TestColmnarMap(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			Compression: &clickhouse.Compression{
				Method: clickhouse.CompressionLZ4,
			},
			//Debug: true,
		})
	)
	if assert.NoError(t, err) {
		if err := checkMinServerVersion(conn, 21, 9); err != nil {
			t.Skip(err.Error())
			return
		}
		const ddl = `
		CREATE TABLE test_map (
			  Col1 Map(String, UInt64)
			, Col2 Map(String, UInt64)
			, Col3 Map(String, UInt64)
		) Engine Memory
		`
		if err := conn.Exec(ctx, "DROP TABLE IF EXISTS test_map"); assert.NoError(t, err) {
			if err := conn.Exec(ctx, ddl); assert.NoError(t, err) {
				if batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_map"); assert.NoError(t, err) {
					var (
						col1Data = []map[string]uint64{}
						col2Data = []map[string]uint64{}
						col3Data = []map[string]uint64{}
					)
					for i := 0; i < 100; i++ {
						col1Data = append(col1Data, map[string]uint64{
							fmt.Sprintf("key_col_1_%d_1", i): uint64(i),
							fmt.Sprintf("key_col_1_%d_2", i): uint64(i),
						})
						col2Data = append(col2Data, map[string]uint64{
							fmt.Sprintf("key_col_2_%d_1", i): uint64(i),
							fmt.Sprintf("key_col_2_%d_2", i): uint64(i),
						})
						col3Data = append(col3Data, map[string]uint64{})
					}
					if err := batch.Column(0).Append(col1Data); !assert.NoError(t, err) {
						return
					}
					if err := batch.Column(1).Append(col2Data); !assert.NoError(t, err) {
						return
					}
					if err := batch.Column(2).Append(col3Data); !assert.NoError(t, err) {
						return
					}

					if assert.NoError(t, batch.Send()) {
						var (
							col1     map[string]uint64
							col2     map[string]uint64
							col3     map[string]uint64
							col1Data = map[string]uint64{
								"key_col_1_10_1": 10,
								"key_col_1_10_2": 10,
							}
							col2Data = map[string]uint64{
								"key_col_2_10_1": 10,
								"key_col_2_10_2": 10,
							}
							col3Data = map[string]uint64{}
						)
						if err := conn.QueryRow(ctx, "SELECT * FROM test_map WHERE Col1['key_col_1_10_1'] = $1", 10).Scan(&col1, &col2, &col3); assert.NoError(t, err) {
							assert.Equal(t, col1Data, col1)
							assert.Equal(t, col2Data, col2)
							assert.Equal(t, col3Data, col3)
						}
					}
				}
			}
		}
	}
}

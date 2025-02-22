package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestQueryRowScanStruct(t *testing.T) {
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
		var result struct {
			Col1 string `ch:"col1"`
			Col2 uint8  `ch:"col2"`
			Col3 *uint8 `ch:"col3"`
			Col4 *uint8 `ch:"col4"`
		}
		if err := conn.QueryRow(ctx, "SELECT 'ABC' AS col1, 42 AS col2, 5 AS col3, NULL AS col4").ScanStruct(&result); assert.NoError(t, err) {
			if assert.Nil(t, result.Col4) {
				assert.Equal(t, "ABC", result.Col1)
				assert.Equal(t, uint8(42), result.Col2)
				assert.Equal(t, uint8(5), *result.Col3)
			}
		}
	}
}
func TestQueryScanStruct(t *testing.T) {
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
		rows, err := conn.Query(ctx, "SELECT number, 'ABC_' || CAST(number AS String) AS col1, now() AS time FROM system.numbers LIMIT 5")
		if assert.NoError(t, err) {
			var i uint64
			for rows.Next() {
				var result struct {
					Col1 uint64    `ch:"number"`
					Col2 string    `ch:"col1"`
					Col3 time.Time `ch:"time"`
				}
				if assert.NoError(t, rows.ScanStruct(&result)) {
					assert.Equal(t, i, result.Col1)
					assert.Equal(t, fmt.Sprintf("ABC_%d", i), result.Col2)
				}
				i++
			}
			if assert.NoError(t, rows.Close()) {
				assert.NoError(t, rows.Err())
			}
		}
	}
}

func TestSelectScanStruct(t *testing.T) {
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
		var result []struct {
			Col1 uint64     `ch:"number"`
			Col2 string     `ch:"col1"`
			Col3 *time.Time `ch:"time"`
		}
		err := conn.Select(ctx, &result, "SELECT number, 'ABC_' || CAST(number AS String) AS col1, now() AS time FROM system.numbers LIMIT 5")
		if assert.NoError(t, err) && assert.Len(t, result, 5) {
			for i, v := range result {
				if assert.NotNil(t, v.Col3) {
					assert.Equal(t, uint64(i), v.Col1)
					assert.Equal(t, fmt.Sprintf("ABC_%d", i), v.Col2)
				}
			}
		}
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

func example() error {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			//Debug:           true,
			DialTimeout:     time.Second,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
		})
	)
	if err != nil {
		return err
	}
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			  Col1 UInt8
			, Col2 String
			, Col3 FixedString(3)
			, Col4 UUID
			, Col5 Map(String, UInt8)
			, Col6 Array(String)
			, Col7 Tuple(String, UInt8, Array(Map(String, String)))
			, Col8 DateTime
		) Engine = Null
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	for i := 0; i < 500_000; i++ {
		err := batch.Append(
			uint8(42),
			"ClickHouse", "Inc",
			uuid.New(),
			map[string]uint8{"key": 1},             // Map(String, UInt8)
			[]string{"Q", "W", "E", "R", "T", "Y"}, // Array(String)
			[]interface{}{ // Tuple(String, UInt8, Array(Map(String, String)))
				"String Value", uint8(5), []map[string]string{
					map[string]string{"key": "value"},
					map[string]string{"key": "value"},
					map[string]string{"key": "value"},
				},
			},
			time.Now(),
		)
		if err != nil {
			return err
		}
	}
	return batch.Send()
}

func main() {
	start := time.Now()
	if err := example(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(time.Since(start))
}

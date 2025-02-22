package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

const ddl = `
CREATE TABLE benchmark (
	  Col1 UInt64
	, Col2 String
	, Col3 Array(UInt8)
	, Col4 DateTime
) Engine Null
`

func benchmark(conn clickhouse.Conn) error {
	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO benchmark")
	if err != nil {
		return err
	}
	var (
		col1 []uint64
		col2 []string
		col3 [][]uint8
		col4 []time.Time
	)
	for i := 0; i < 1_000_000; i++ {
		col1 = append(col1, uint64(i))
		col2 = append(col2, "Golang SQL database driver")
		col3 = append(col3, []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9})
		col4 = append(col4, time.Now())
	}
	if err := batch.Column(0).Append(col1); err != nil {
		return err
	}
	if err := batch.Column(1).Append(col2); err != nil {
		return err
	}
	if err := batch.Column(2).Append(col3); err != nil {
		return err
	}
	if err := batch.Column(3).Append(col4); err != nil {
		return err
	}
	return batch.Send()
}

func main() {
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
		log.Fatal(err)
	}
	if err := conn.Exec(ctx, "DROP TABLE IF EXISTS benchmark"); err != nil {
		log.Fatal(err)
	}
	if err := conn.Exec(ctx, ddl); err != nil {
		log.Fatal(err)
	}
	start := time.Now()
	if err := benchmark(conn); err != nil {
		log.Fatal(err)
	}
	fmt.Println(time.Since(start))
}

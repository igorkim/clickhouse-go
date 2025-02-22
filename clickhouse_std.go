package clickhouse

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
)

func init() {
	sql.Register("clickhouse", &stdDriver{})
}

func OpenDB(opt *Options) *sql.DB {
	var settings []string
	if opt.MaxIdleConns > 0 {
		settings = append(settings, "SetMaxIdleConns")
	}
	if opt.MaxOpenConns > 0 {
		settings = append(settings, "SetMaxOpenConns")
	}
	if opt.ConnMaxLifetime > 0 {
		settings = append(settings, "SetConnMaxLifetime")
	}
	if len(settings) != 0 {
		return sql.OpenDB(&stdDriver{
			err: fmt.Errorf("cannot connect. invalid settings. use %s (see https://pkg.go.dev/database/sql)", strings.Join(settings, ",")),
		})
	}
	opt.setDefaults()
	return sql.OpenDB(&stdDriver{
		opt: opt,
	})
}

type stdDriver struct {
	err    error
	opt    *Options
	conn   *connect
	commit func() error
	connID int64
}

func (d *stdDriver) Driver() driver.Driver {
	return d
}

func (d *stdDriver) Connect(context.Context) (driver.Conn, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.Open("")
}

func (d *stdDriver) Open(dsn string) (_ driver.Conn, err error) {
	var (
		conn   *connect
		connID = int(atomic.AddInt64(&d.connID, 1))
	)
	if d.opt == nil {
		d.opt = &Options{}
		if err = d.opt.fromDSN(dsn); err != nil {
			return nil, err
		}
	}
	for num := range d.opt.Addr {
		if d.opt.ConnOpenStrategy == ConnOpenRoundRobin {
			num = int(connID) % len(d.opt.Addr)
		}
		if conn, err = dial(d.opt.Addr[num], connID, d.opt); err == nil {
			return &stdDriver{
				conn: conn,
			}, nil
		}
	}
	return nil, err
}

func (std *stdDriver) ResetSession(ctx context.Context) error {
	if std.conn.isBad() {
		return driver.ErrBadConn
	}
	return nil
}

func (std *stdDriver) Ping(ctx context.Context) error { return std.conn.ping(ctx) }

func (std *stdDriver) Begin() (driver.Tx, error) { return std, nil }

func (std *stdDriver) Commit() error {
	if std.commit == nil {
		return nil
	}
	defer func() {
		std.commit = nil
	}()
	return std.commit()
}

func (std *stdDriver) Rollback() error {
	std.commit = nil
	std.conn.close()
	return nil
}

func (std *stdDriver) CheckNamedValue(nv *driver.NamedValue) error { return nil }

func (std *stdDriver) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := std.conn.exec(ctx, query, rebind(args)...); err != nil {
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (std *stdDriver) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	r, err := std.conn.query(ctx, query, rebind(args)...)
	if err != nil {
		return nil, err
	}
	return &stdRows{
		rows: r,
	}, nil
}

func (std *stdDriver) Prepare(query string) (driver.Stmt, error) {
	return std.PrepareContext(context.Background(), query)
}

func (std *stdDriver) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	batch, err := std.conn.prepareBatch(ctx, query, func(c *connect) {})
	if err != nil {
		return nil, err
	}
	std.commit = batch.Send
	return &stdBatch{
		batch: batch,
	}, nil
}

func (std *stdDriver) Close() error { return std.conn.close() }

type stdBatch struct {
	batch *batch
}

func (s *stdBatch) NumInput() int { return -1 }
func (s *stdBatch) Exec(args []driver.Value) (driver.Result, error) {
	values := make([]interface{}, 0, len(args))
	for _, v := range args {
		values = append(values, v)
	}
	if err := s.batch.Append(values...); err != nil {
		return nil, err
	}
	return driver.RowsAffected(0), nil
}

func (s *stdBatch) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	values := make([]driver.Value, 0, len(args))
	for _, v := range args {
		values = append(values, v.Value)
	}
	return s.Exec(values)
}

func (s *stdBatch) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("only Exec method supported in batch mode")
}

func (s *stdBatch) Close() error { return nil }

type stdRows struct {
	rows *rows
}

func (r *stdRows) Columns() []string {
	return r.rows.Columns()
}

func (r *stdRows) ColumnTypeScanType(idx int) reflect.Type {
	return r.rows.block.Columns[idx].ScanType()
}

func (r *stdRows) ColumnTypeDatabaseTypeName(idx int) string {
	return string(r.rows.block.Columns[idx].Type())
}

func (r *stdRows) ColumnTypeNullable(idx int) (nullable, ok bool) {
	_, ok = r.rows.block.Columns[idx].(*column.Nullable)
	return ok, true
}

func (r *stdRows) ColumnTypePrecisionScale(idx int) (precision, scale int64, ok bool) {
	switch col := r.rows.block.Columns[idx].(type) {
	case *column.Decimal:
		return col.Precision(), col.Scale(), true
	case interface{ Base() column.Interface }:
		switch col := col.Base().(type) {
		case *column.Decimal:
			return col.Precision(), col.Scale(), true
		}
	}
	return 0, 0, false
}

func (r *stdRows) Next(dest []driver.Value) error {
	if len(r.rows.block.Columns) != len(dest) {
		return &OpError{
			Op:  "Next",
			Err: fmt.Errorf("expected %d destination arguments in Next, not %d", len(r.rows.block.Columns), len(dest)),
		}
	}
	if r.rows.Next() {
		for i := range dest {
			nullable, ok := r.ColumnTypeNullable(i)
			switch value := r.rows.block.Columns[i].Row(r.rows.row-1, nullable && ok).(type) {
			case driver.Valuer:
				v, err := value.Value()
				if err != nil {
					return err
				}
				dest[i] = v
			default:
				dest[i] = value
			}
		}
		return nil
	}
	if err := r.rows.Err(); err != nil {
		return err
	}
	return io.EOF
}

func (r *stdRows) HasNextResultSet() bool {
	return r.rows.totals != nil
}

func (r *stdRows) NextResultSet() error {
	switch {
	case r.rows.totals != nil:
		r.rows.block = r.rows.totals
		r.rows.totals = nil
	default:
		return io.EOF
	}
	return nil
}

func (r *stdRows) Close() error {
	return r.rows.Close()
}

package column

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/binary"
	"github.com/ClickHouse/clickhouse-go/v2/lib/timezone"
)

var (
	minDateTime, _ = time.Parse("2006-01-02 15:04:05", "1970-01-01 00:00:00")
	maxDateTime, _ = time.Parse("2006-01-02 15:04:05", "2105-12-31 23:59:59")
)

type DateTime struct {
	chType   Type
	values   Int32
	timezone *time.Location
}

func (dt *DateTime) parse(t Type) (_ *DateTime, err error) {
	if dt.chType = t; dt.chType == "DateTime" {
		return dt, nil
	}
	var name = strings.TrimSuffix(strings.TrimPrefix(string(t), "DateTime('"), "')")
	if dt.timezone, err = timezone.Load(name); err != nil {
		return nil, err
	}
	return dt, nil
}

func (dt *DateTime) Type() Type {
	return dt.chType
}

func (col *DateTime) ScanType() reflect.Type {
	return scanTypeTime
}

func (dt *DateTime) Rows() int {
	return len(dt.values)
}

func (dt *DateTime) Row(i int, ptr bool) interface{} {
	value := dt.row(i)
	if ptr {
		return &value
	}
	return value
}

func (dt *DateTime) ScanRow(dest interface{}, row int) error {
	switch d := dest.(type) {
	case *time.Time:
		*d = dt.row(row)
	case **time.Time:
		*d = new(time.Time)
		**d = dt.row(row)
	default:
		return &ColumnConverterError{
			Op:   "ScanRow",
			To:   fmt.Sprintf("%T", dest),
			From: "DateTime",
		}
	}
	return nil
}

func (dt *DateTime) Append(v interface{}) (nulls []uint8, err error) {
	switch v := v.(type) {
	case []time.Time:
		in := make([]int32, 0, len(v))
		for _, t := range v {
			if err := dateOverflow(minDateTime, maxDateTime, t, "2006-01-02 15:04:05"); err != nil {
				return nil, err
			}
			in = append(in, int32(t.Unix()))
		}
		dt.values, nulls = append(dt.values, in...), make([]uint8, len(v))
	case []*time.Time:
		nulls = make([]uint8, len(v))
		for i, v := range v {
			switch {
			case v != nil:
				if err := dateOverflow(minDateTime, maxDateTime, *v, "2006-01-02 15:04:05"); err != nil {
					return nil, err
				}
				dt.values = append(dt.values, int32(v.Unix()))
			default:
				dt.values, nulls[i] = append(dt.values, 0), 1
			}
		}
	default:
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   "DateTime",
			From: fmt.Sprintf("%T", v),
		}
	}
	return
}

func (dt *DateTime) AppendRow(v interface{}) error {
	var datetime int32
	switch v := v.(type) {
	case time.Time:
		if err := dateOverflow(minDateTime, maxDateTime, v, "2006-01-02 15:04:05"); err != nil {
			return err
		}
		datetime = int32(v.Unix())
	case *time.Time:
		if v != nil {
			if err := dateOverflow(minDateTime, maxDateTime, *v, "2006-01-02 15:04:05"); err != nil {
				return err
			}
			datetime = int32(v.Unix())
		}
	case nil:
	default:
		return &ColumnConverterError{
			Op:   "AppendRow",
			To:   "DateTime",
			From: fmt.Sprintf("%T", v),
		}
	}
	dt.values = append(dt.values, datetime)
	return nil
}

func (dt *DateTime) Decode(decoder *binary.Decoder, rows int) error {
	return dt.values.Decode(decoder, rows)
}

func (dt *DateTime) Encode(encoder *binary.Encoder) error {
	return dt.values.Encode(encoder)
}

func (dt *DateTime) row(i int) time.Time {
	v := time.Unix(int64(dt.values[i]), 0)
	if dt.timezone != nil {
		v = v.In(dt.timezone)
	}
	return v
}

var _ Interface = (*DateTime)(nil)

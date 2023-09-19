package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func Bool(v bool) sql.NullBool {
	return sql.NullBool{Bool: v, Valid: true}
}

func Int32(v int32) sql.NullInt32 {
	return sql.NullInt32{Int32: v, Valid: true}
}

func Int64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: true}
}

func String(v string) sql.NullString {
	return sql.NullString{String: v, Valid: true}
}

func NullUUID(v uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: v, Valid: true}
}

func NullTime(v time.Time) sql.NullInt32 {
	return Int32(int32(v.Unix()))
}

func Time(v time.Time) int32 {
	return int32(v.Unix())
}

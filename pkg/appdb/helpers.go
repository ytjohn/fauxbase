package appdb

import (
	"database/sql"
	"time"
)

// Add at package level
type contextKey string

const (
	DbContextKey contextKey = "database"
)

func NullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func NullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

func NullInt32(i int32) sql.NullInt32 {
	return sql.NullInt32{Int32: i, Valid: true}
}

func NullBool(b bool) sql.NullBool {
	return sql.NullBool{Bool: b, Valid: true}
}

func StringToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func DateToNullDate(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

func BoolToNullBool(b bool) sql.NullBool {
	return sql.NullBool{Bool: b, Valid: true}
}

func TimeToNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

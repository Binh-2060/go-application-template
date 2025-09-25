package db

import "github.com/Masterminds/squirrel"

func GetPSQLCommand() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
}

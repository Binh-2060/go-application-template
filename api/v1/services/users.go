package services

import (
	"context"

	dbquery "github.com/Binh-2060/go-application-template/pkg/db-pkg/db-query"
)

func GetUserService(ctx context.Context) (any, error) {
	result, err := dbquery.GetUserDbQuery(ctx)
	if err != nil {
		return result, err
	}

	return result, nil
}

package dbquery

import (
	"context"

	"github.com/Binh-2060/go-application-template/config/db"
	dbpkg "github.com/Binh-2060/go-application-template/pkg/db-pkg"
	dbschema "github.com/Binh-2060/go-application-template/pkg/db-pkg/db-schema"
)

func GetUserDbQuery(ctx context.Context) ([]dbschema.GetUserDbSchema, error) {
	var res = []dbschema.GetUserDbSchema{}
	psql := db.GetPSQLCommand()
	query := psql.
		Select("id, name, email").
		From("users")
	sql, args, err := query.ToSql()
	if err != nil {
		return res, err
	}

	rows, err := dbpkg.DB.Query(ctx, sql, args...)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var item dbschema.GetUserDbSchema
		err = rows.Scan(&item.ID, &item.Name, &item.Email)
		if err != nil {
			return res, err
		}

		res = append(res, item)
	}

	return res, nil
}

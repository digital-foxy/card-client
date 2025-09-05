package loader

import (
	"context"
	dsql "database/sql"
	"database/sql/driver"
	"fmt"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/pkg/errors"
	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/store"
	"github.com/r3dpixel/card-client/services/vault"
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
	"modernc.org/sqlite"
)

type sqliteDriver struct {
	*sqlite.Driver
}

func (d sqliteDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return conn, err
	}
	c := conn.(interface {
		Exec(stmt string, args []driver.Value) (driver.Result, error)
	})
	if _, err := c.Exec("PRAGMA foreign_keys = on;", nil); err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "failed to enable enable foreign keys")
	}
	return conn, nil
}

func init() {
	dsql.Register("sqlite3", sqliteDriver{Driver: &sqlite.Driver{}})
}

type Service struct {
	options      opts.StoreOptions
	vaultService vault.Service
	provider     store.Provider
}

func NewService(
	opts opts.StoreOptions,
	vaultService vault.Service,
	provider store.Provider,
) *Service {
	return &Service{
		options:      opts,
		vaultService: vaultService,
		provider:     provider,
	}
}

func (l *Service) LoadVault(vault string) (store.Service, error) {
	v, ok := l.vaultService.GetVault(vault)
	if !ok {
		return nil, trace.Err().
			Field(trace.SERVICE, "loader").
			Field("vault", vault).
			Msg(fmt.Sprintf("Vault not found"))
	}
	freshDB := !filex.PathExists(v.DbFilePath)
	dsn := fmt.Sprintf("file:%s?_fk=1", v.DbFilePath)

	entDriver, err := sql.Open(dialect.SQLite, dsn)
	if err != nil {
		return nil, err
	}

	db := entDriver.DB()
	db.SetMaxOpenConns(l.options.DbOptions.MaxConnections)
	db.SetMaxIdleConns(l.options.DbOptions.IdleConnections)
	db.SetConnMaxLifetime(l.options.DbOptions.MaxLifetime)

	client := ent.NewClient(ent.Driver(entDriver))

	if err = client.Schema.Create(context.Background()); err != nil {
		_ = client.Close()
		return nil, err
	}

	storeService := l.provider(client, v, l.options.PngOptions)
	if freshDB {
		err = storeService.InsertStandardTags(context.Background())
		if err != nil {
			log.Warn().Err(err).
				Str(trace.SERVICE, "loader").
				Str("vault", v.Name).
				Msg("Failed to insert staging tags")
		}
	}

	return storeService, nil
}

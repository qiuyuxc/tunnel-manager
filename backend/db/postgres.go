package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// openPostgres connects to PostgreSQL through pgx, wrapped so the store's
// SQLite-style ? placeholders keep working.
func openPostgres(dsn string) (*sql.DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	handle := sql.OpenDB(placeholderConnector{inner: stdlib.GetConnector(*config)})
	if err := handle.Ping(); err != nil {
		handle.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return handle, nil
}

// placeholderConnector decorates a driver.Connector so every statement sent
// through it has its ? placeholders rewritten to PostgreSQL's $N form.
type placeholderConnector struct {
	inner driver.Connector
}

func (c placeholderConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return placeholderConn{Conn: conn}, nil
}

func (c placeholderConnector) Driver() driver.Driver {
	return c.inner.Driver()
}

// placeholderConn forwards every optional driver capability the underlying
// connection implements, rewriting query text on the way through.
type placeholderConn struct {
	driver.Conn
}

func (c placeholderConn) Prepare(query string) (driver.Stmt, error) {
	return c.Conn.Prepare(RewritePlaceholders(query))
}

func (c placeholderConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	query = RewritePlaceholders(query)
	if prepare, ok := c.Conn.(driver.ConnPrepareContext); ok {
		return prepare.PrepareContext(ctx, query)
	}
	return c.Conn.Prepare(query)
}

func (c placeholderConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if begin, ok := c.Conn.(driver.ConnBeginTx); ok {
		return begin.BeginTx(ctx, opts)
	}
	return nil, driver.ErrSkip
}

func (c placeholderConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.ExecContext(ctx, RewritePlaceholders(query), args)
}

func (c placeholderConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.QueryContext(ctx, RewritePlaceholders(query), args)
}

func (c placeholderConn) Ping(ctx context.Context) error {
	if pinger, ok := c.Conn.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return nil
}

func (c placeholderConn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.Conn.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}
	return nil
}

func (c placeholderConn) CheckNamedValue(value *driver.NamedValue) error {
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return driver.ErrSkip
}

func (c placeholderConn) IsValid() bool {
	if validator, ok := c.Conn.(driver.Validator); ok {
		return validator.IsValid()
	}
	return true
}

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

/*
Querier is the read/write surface shared by *pgxpool.Pool and pgx.Tx.

Repositories should accept a Querier rather than the pool directly, so the same
code runs inside or outside a transaction without a second method set.
*/
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, table pgx.Identifier, columns []string, src pgx.CopyFromSource) (int64, error)
}

// beginner is satisfied by both *pgxpool.Pool and pgx.Tx. On a Tx, Begin opens
// a savepoint, which is what makes nested ExecTx calls work.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type txKey struct{}

/*
Q returns the Querier bound to ctx: the in-flight transaction if ExecTx put one
there, otherwise the shared pool.

Handlers and repositories call this instead of Pool(), so a function written for
a plain query automatically joins an enclosing transaction.
*/
func Q(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return Pool()
}

/*
Run fn inside a transaction, committing if it returns nil and rolling back
otherwise.

The tx is stashed on the context passed to fn, so anything downstream calling
Q(ctx) participates in it. A nested ExecTx opens a savepoint rather than a
second transaction, so an inner failure unwinds only the inner work.

Rollback runs on a context detached from ctx's cancellation — otherwise an
aborted request would cancel its own cleanup and leave the connection to be
reset by the pool. A panic in fn rolls back and re-panics.
*/
func ExecTx(ctx context.Context, fn func(ctx context.Context, q Querier) error) error {
	return execTx(ctx, nil, fn)
}

/*
ExecTx with explicit isolation / access mode, e.g.:

	db.ExecTxOptions(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable}, fn)

Options apply only to an outermost transaction; a nested call is a savepoint and
inherits the outer settings.
*/
func ExecTxOptions(ctx context.Context, opts pgx.TxOptions, fn func(ctx context.Context, q Querier) error) error {
	return execTx(ctx, &opts, fn)
}

func execTx(ctx context.Context, opts *pgx.TxOptions, fn func(ctx context.Context, q Querier) error) (err error) {
	var tx pgx.Tx

	switch outer := Q(ctx).(type) {
	case pgx.Tx:
		// Nested: savepoint on the existing tx. TxOptions are not applicable.
		tx, err = outer.Begin(ctx)
	default:
		b, ok := outer.(beginner)
		if !ok {
			return fmt.Errorf("db: %T cannot begin a transaction", outer)
		}
		if opts != nil {
			if bt, ok := b.(interface {
				BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
			}); ok {
				tx, err = bt.BeginTx(ctx, *opts)
				break
			}
		}
		tx, err = b.Begin(ctx)
	}
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}

	// Rollback must survive the caller's context being cancelled.
	rollbackCtx := context.WithoutCancel(ctx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(rollbackCtx)
			panic(p)
		}
		if err != nil {
			// Rollback error is subordinate to the error that caused it.
			_ = tx.Rollback(rollbackCtx)
		}
	}()

	if err = fn(context.WithValue(ctx, txKey{}, tx), tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}

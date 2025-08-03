package ctxs

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const TxKey = "pgxTxKey"

func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

func Tx(ctx context.Context) (pgx.Tx, bool) {
	val := ctx.Value(TxKey)
	if val == nil {
		return nil, false
	}

	tx, ok := val.(pgx.Tx)
	return tx, ok
}

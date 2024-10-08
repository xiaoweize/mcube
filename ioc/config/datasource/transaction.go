package datasource

import (
	"context"

	"gorm.io/gorm"
)

type TransactionCtxKey struct{}

func WithTransactionCtx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, TransactionCtxKey{}, tx)
}

func GetTransactionFromCtx(ctx context.Context) *gorm.DB {
	if ctx != nil {
		tx, ok := ctx.Value(TransactionCtxKey{}).(*gorm.DB)
		if ok {
			return tx
		}
	}

	return nil
}

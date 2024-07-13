// data.go
package data

import (
	"codewave-timer/codewaveTimer/internal/biz"
	"codewave-timer/codewaveTimer/pkg/cache"
	"context"
	"gorm.io/gorm"
)

type contextTxKey struct{}

func (d *Data) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.Db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}

func (d *Data) DB(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return d.Db
}

func NewTransaction(d *Data) biz.Transaction {
	return d
}

type Data struct {
	Db    *gorm.DB
	Cache *cache.Client
}

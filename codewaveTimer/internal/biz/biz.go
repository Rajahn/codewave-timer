// biz.go
package biz

import "context"

type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}

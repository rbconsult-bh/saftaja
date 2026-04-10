package store

import (
	"github.com/jackc/pgx/v5"
)

type TransactionQuerier interface {
	Querier
	WithTx(tx pgx.Tx) TransactionQuerier
}

type txQuerier struct {
	*Queries
}

func (t *txQuerier) WithTx(tx pgx.Tx) TransactionQuerier {
	return &txQuerier{Queries: t.Queries.WithTx(tx)}
}

func NewTransactionQuerier(db DBTX) TransactionQuerier {
	return &txQuerier{Queries: New(db)}
}

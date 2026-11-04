package models

import "github.com/govalues/decimal"

type Balance struct {
	ID          int64           `json:"-"`
	Account_id  int64           `json:"-"`
	Amount      decimal.Decimal `json:"amount"`
	Currency_id int64           `json:"-"`
	Currency    Currency        `json:"currency"`
}

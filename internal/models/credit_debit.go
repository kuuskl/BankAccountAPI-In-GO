package models

import "github.com/govalues/decimal"

type CreditDebit struct {
	Currency string          `json:"currency"`
	Amount   decimal.Decimal `json:"amount"`
}

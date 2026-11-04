package models

type Currency struct {
	ID           int64  `json:"-"`
	CurrencyCode string `json:"currency_code"`
}

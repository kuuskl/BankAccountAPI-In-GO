package models

import "github.com/govalues/decimal"

type ExchangeRates struct {
	ID    int64           `json:"id"`
	Fromc int64           `json:"fromc"`
	Toc   int64           `json:"toc"`
	Rate  decimal.Decimal `json:"rate"`
}

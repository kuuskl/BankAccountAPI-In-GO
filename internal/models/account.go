package models

type Account struct {
	ID       int64     `json:"-"`
	IBAN     string    `json:"iban"`
	Name     string    `json:"name"`
	Balances []Balance `json:"balances"`
}

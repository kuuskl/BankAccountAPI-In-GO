package repository

import (
	"context"
	"database/sql"
	"errors"

	"go-mysql-api/internal/models"

	"github.com/govalues/decimal"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Accounts(ctx context.Context) ([]models.Account, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, iban, name FROM account ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var account models.Account
		if err := rows.Scan(&account.ID, &account.IBAN, &account.Name); err != nil {
			return nil, err
		}
		account.Balances, err = r.getBalances(ctx, account)
		if err != nil {
			return []models.Account{}, err
		}
		accounts = append(accounts, account)

	}

	return accounts, rows.Err()
}

func (r *AccountRepository) GetAccountByIban(ctx context.Context, iban string) (models.Account, error) {
	var account models.Account
	err := r.db.QueryRowContext(ctx, "SELECT id, iban, name FROM account WHERE iban = ?", iban).
		Scan(&account.ID, &account.IBAN, &account.Name)

	if errors.Is(err, sql.ErrNoRows) {
		return models.Account{}, err
	}

	account.Balances, err = r.getBalances(ctx, account)
	if err != nil {
		return models.Account{}, err
	}

	return account, err
}

func (r *AccountRepository) UpdateBalance(ctx context.Context, balance models.Balance, amount decimal.Decimal) error {
	result, err := r.db.ExecContext(
		ctx,
		"UPDATE balance SET amount = ? WHERE id = ?",
		amount,
		balance.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *AccountRepository) AddBalance(ctx context.Context, account models.Account, currency models.Currency, amount decimal.Decimal) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO balance (account_id, amount, currency_id) VALUES (?, ?, ?)`, account.ID, amount, currency.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *AccountRepository) GetExchangeRate(ctx context.Context, fromCurrency int64, toCurrency int64) (models.ExchangeRates, error) {
	var rate models.ExchangeRates

	err := r.db.QueryRowContext(ctx, "SELECT id, fromc_id, toc_id, rate FROM exchange_rates WHERE fromc_id = ? and toc_id = ?", fromCurrency, toCurrency).
		Scan(&rate.ID, &rate.Fromc, &rate.Toc, &rate.Rate)

	if errors.Is(err, sql.ErrNoRows) {
		return models.ExchangeRates{}, err
	}

	return rate, err
}

/* Internal methods */
func (r *AccountRepository) getBalances(ctx context.Context, account models.Account) ([]models.Balance, error) {
	var balances []models.Balance
	rows, err := r.db.QueryContext(ctx, "SELECT id, account_id, amount, currency_id FROM balance WHERE account_id = ?", account.ID)
	if err != nil {
		return []models.Balance{}, err
	}

	for rows.Next() {
		var balance models.Balance
		err = rows.Scan(&balance.ID, &balance.Account_id, &balance.Amount, &balance.Currency_id)
		if err != nil {
			return []models.Balance{}, err
		}
		balance.Currency, err = r.getCurrency(ctx, balance)
		if err != nil {
			return []models.Balance{}, err
		}

		balances = append(balances, balance)
	}
	defer rows.Close()

	return balances, rows.Err()
}

func (r *AccountRepository) getCurrency(ctx context.Context, balance models.Balance) (models.Currency, error) {
	var currency models.Currency
	err := r.db.QueryRowContext(ctx, "SELECT id, currency_code FROM currency WHERE id = ?", balance.Currency_id).
		Scan(&currency.ID, &currency.CurrencyCode)

	if errors.Is(err, sql.ErrNoRows) {
		return models.Currency{}, err
	}

	return currency, err
}

func (r *AccountRepository) GetCurrencyByCurrencyCode(ctx context.Context, code string) (models.Currency, error) {
	var currency models.Currency
	err := r.db.QueryRowContext(ctx, "SELECT id, currency_code FROM currency WHERE currency_code = ?", code).
		Scan(&currency.ID, &currency.CurrencyCode)

	if errors.Is(err, sql.ErrNoRows) {
		return models.Currency{}, err
	}

	return currency, err
}

/* Methods for data testing */
func (r *AccountRepository) Balances(ctx context.Context) ([]models.Balance, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, account_id, amount, currency_id FROM balance ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ms []models.Balance
	for rows.Next() {
		var m models.Balance
		if err := rows.Scan(&m.ID, &m.Account_id, &m.Amount, &m.Currency_id); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}

	return ms, rows.Err()
}

func (r *AccountRepository) Currencys(ctx context.Context) ([]models.Currency, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, currency_code FROM currency ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ms []models.Currency
	for rows.Next() {
		var m models.Currency
		if err := rows.Scan(&m.ID, &m.CurrencyCode); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}

	return ms, rows.Err()
}

func (r *AccountRepository) ExchangeRates(ctx context.Context) ([]models.ExchangeRates, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, fromc_id, toc_id, rate FROM exchange_rates ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ms []models.ExchangeRates
	for rows.Next() {
		var m models.ExchangeRates
		if err := rows.Scan(&m.ID, &m.Fromc, &m.Toc, &m.Rate); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}

	return ms, rows.Err()
}

package repository

import (
	"context"
	"errors"
	"go-mysql-api/internal/models"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/govalues/decimal"
)

func TestAccountsRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	accountRows := sqlmock.NewRows([]string{"id", "iban", "name"}).
		AddRow(1, "EE120000012345678901", "Account One").
		AddRow(2, "EE120000012345678901", "Account Two")

	mock.ExpectQuery("SELECT id, iban, name FROM account ORDER BY id").
		WillReturnRows(accountRows)

	balanceRows := sqlmock.NewRows([]string{"id", "account_id", "amount", "currency_id"}).
		AddRow(1, 1, 100, 1)
	mock.ExpectQuery("SELECT id, account_id, amount, currency_id FROM balance WHERE account_id = \\?").WithArgs(int64(1)).
		WillReturnRows(balanceRows)

	currencyRows := sqlmock.NewRows([]string{"id", "currency_code"}).
		AddRow(1, "EUR")
	mock.ExpectQuery("SELECT id, currency_code FROM currency WHERE id = \\?").WithArgs(int64(1)).
		WillReturnRows(currencyRows)

	balanceRows2 := sqlmock.NewRows([]string{"id", "account_id", "amount", "currency_id"}).
		AddRow(1, 2, 200, 1)
	mock.ExpectQuery("SELECT id, account_id, amount, currency_id FROM balance WHERE account_id = \\?").WithArgs(int64(2)).
		WillReturnRows(balanceRows2)

	currencyRows2 := sqlmock.NewRows([]string{"id", "currency_code"}).
		AddRow(1, "EUR")
	mock.ExpectQuery("SELECT id, currency_code FROM currency WHERE id = \\?").WithArgs(int64(1)).
		WillReturnRows(currencyRows2)

	repo := NewAccountRepository(db)

	accounts, err := repo.Accounts(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	if accounts[0].Name != "Account One" {
		t.Fatalf("expected first name Account One, got %s", accounts[0].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	if accounts[0].Balances[0].Currency.CurrencyCode != "EUR" {
		t.Fatalf("expected EUR balance, got %s", accounts[0].Balances[0].Currency.CurrencyCode)
	}

}

func TestAccountsRepository_GetAccountByIban(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	accountRows := sqlmock.NewRows([]string{"id", "iban", "name"}).
		AddRow(1, "EE120000012345678901", "Account One")

	mock.ExpectQuery("SELECT id, iban, name FROM account WHERE iban = \\?").WithArgs(string("EE120000012345678901")).
		WillReturnRows(accountRows)

	balanceRows := sqlmock.NewRows([]string{"id", "account_id", "amount", "currency_id"}).
		AddRow(1, 1, 100, 1)
	mock.ExpectQuery("SELECT id, account_id, amount, currency_id FROM balance WHERE account_id = \\?").WithArgs(int64(1)).
		WillReturnRows(balanceRows)

	currencyRows := sqlmock.NewRows([]string{"id", "currency_code"}).
		AddRow(1, "EUR")
	mock.ExpectQuery("SELECT id, currency_code FROM currency WHERE id = \\?").WithArgs(int64(1)).
		WillReturnRows(currencyRows)

	repo := NewAccountRepository(db)

	account, err := repo.GetAccountByIban(context.Background(), "EE120000012345678901")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if account.Name != "Account One" {
		t.Fatalf("expected first name Account One, got %s", account.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	if account.Balances[0].Currency.CurrencyCode != "EUR" {
		t.Fatalf("expected EUR balance, got %s", account.Balances[0].Currency.CurrencyCode)
	}

}

func TestAccountsRepository_AddBalance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()
	a, _ := decimal.Parse("79.99")
	mock.ExpectExec("INSERT INTO balance \\(account_id, amount, currency_id\\) VALUES \\(\\?, \\?, \\?\\)").
		WithArgs(1, a, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewAccountRepository(db)

	account := models.Account{
		ID:   1,
		IBAN: "EE120000012345678901",
		Name: "Account One",
	}
	currency := models.Currency{
		ID:           1,
		CurrencyCode: "EUR",
	}
	amount := a

	err = repo.AddBalance(context.Background(), account, currency, amount)
	if err != nil {
		t.Fatalf("AddBalance returned error: %v", err)
	}
}

func TestAccountsRepository_UpdateBalance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	a, _ := decimal.Parse("1099.99")
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE balance SET amount = \\? WHERE id = \\?").
		WithArgs(a, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewAccountRepository(db)

	balance := models.Balance{
		ID:          1,
		Account_id:  1,
		Amount:      a,
		Currency_id: 1,
	}

	b, _ := decimal.Parse("1099.99")
	err = repo.UpdateBalance(context.Background(), balance, b)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

}

func TestAccountsRepository_ExchangeBalancesRollsBackWhenCreditUpdateFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	fromAmount, _ := decimal.Parse("90")
	toAmount, _ := decimal.Parse("17")
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE balance SET amount = \\? WHERE id = \\?").
		WithArgs(fromAmount, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE balance SET amount = \\? WHERE id = \\?").
		WithArgs(toAmount, int64(2)).
		WillReturnError(errors.New("credit update failed"))
	mock.ExpectRollback()

	repo := NewAccountRepository(db)
	err = repo.ExchangeBalances(
		context.Background(),
		models.Account{ID: 1},
		models.Balance{ID: 1},
		fromAmount,
		&models.Balance{ID: 2},
		models.Currency{ID: 2},
		toAmount,
	)
	if err == nil {
		t.Fatal("expected credit update error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAccountsRepository_GetAccountByIban_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "iban", "name"})
	mock.ExpectQuery("SELECT id, iban, name FROM account WHERE iban = \\?").
		WithArgs(string("EE120000012345678901")).
		WillReturnRows(rows)

	repo := NewAccountRepository(db)

	_, err = repo.GetAccountByIban(context.Background(), "EE120000012345678901")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

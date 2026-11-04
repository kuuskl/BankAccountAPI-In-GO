package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-mysql-api/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/govalues/decimal"
)

type fakeAccountRepo struct {
	accounts      []models.Account
	currencys     []models.Currency
	exchangeRates []models.ExchangeRates
	balances      []models.Balance
	err           error
}

func (f *fakeAccountRepo) Accounts(ctx context.Context) ([]models.Account, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.accounts, nil
}

func (f *fakeAccountRepo) AddBalance(ctx context.Context, account models.Account, currency models.Currency, amount decimal.Decimal) error {

	if f.err != nil {
		return f.err
	}

	for i, acc := range f.accounts {
		if acc.IBAN == account.IBAN {
			newBalance := models.Balance{}
			newBalance.ID = int64(len(f.accounts[i].Balances) + 1)
			newBalance.Account_id = account.ID
			newBalance.Amount = amount
			newBalance.Currency_id = currency.ID
			newBalance.Currency = currency
			f.accounts[i].Balances = append(f.accounts[i].Balances, newBalance)
			return nil
		}
	}
	return nil
}

func (f *fakeAccountRepo) GetAccountByIban(ctx context.Context, iban string) (models.Account, error) {
	if f.err != nil {
		return models.Account{}, f.err
	}
	for _, account := range f.accounts {
		if account.IBAN == iban {
			return account, nil
		}
	}

	return models.Account{}, sql.ErrNoRows
}

func (f *fakeAccountRepo) GetCurrencyByCurrencyCode(ctx context.Context, code string) (models.Currency, error) {
	if f.err != nil {
		return models.Currency{}, f.err
	}
	for _, currency := range f.currencys {
		if currency.CurrencyCode == code {
			return currency, nil
		}
	}
	return models.Currency{}, errors.New("not found")
}

func (f *fakeAccountRepo) GetExchangeRate(ctx context.Context, fromCurrency int64, toCurrency int64) (models.ExchangeRates, error) {
	if f.err != nil {
		return models.ExchangeRates{}, f.err
	}
	for _, exchangeRate := range f.exchangeRates {
		if exchangeRate.Fromc == fromCurrency && exchangeRate.Toc == toCurrency {
			return exchangeRate, nil
		}
	}
	return models.ExchangeRates{}, errors.New("not found")
}

func (f *fakeAccountRepo) UpdateBalance(ctx context.Context, balance models.Balance, amount decimal.Decimal) error {
	if f.err != nil {
		return f.err
	}
	for i, account := range f.accounts {
		if account.ID == balance.Account_id {

			for j, acc_balance := range f.accounts[i].Balances {
				if acc_balance.ID == balance.ID {
					f.accounts[i].Balances[j].Amount = amount
					return nil
				}
			}
		}
	}

	return errors.New("Balance found")
}

func TestAccounts_Ok(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	rr := httptest.NewRecorder()

	h.Accounts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var accounts []models.Account
	if err := json.Unmarshal(rr.Body.Bytes(), &accounts); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestGetAccountByIban_Ok(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/accounts/EE120000012345678901", nil)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetAccountByIban(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var account models.Account
	if err := json.Unmarshal(rr.Body.Bytes(), &account); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if account.IBAN != "EE120000012345678901" {
		t.Fatalf("expected iban EE120000012345678901, got %s", account.IBAN)
	}
}

func TestGetAccountByIban_NotFound(t *testing.T) {
	repo := CreateRepoData()

	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/accounts/EE120000012345678903", nil)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678903")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetAccountByIban(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestAddBalance_Ok(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	body := `{"Currency":"USD","Amount":79.99}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.AddToBalance(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	h.GetAccountByIban(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var account models.Account

	if err := json.Unmarshal(rr.Body.Bytes(), &account); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if account.IBAN != "EE120000012345678901" {
		t.Fatalf("expected iban EE120000012345678901, got %s", account.IBAN)
	}

	if len(account.Balances) != 2 {
		t.Fatalf("expected two balances, got %d", len(account.Balances))
	}

	aNew, _ := decimal.Parse("79.99")
	if account.Balances[1].Amount != aNew {
		t.Fatalf("USD balance has wrong amount, got %v", account.Balances[1].Amount)
	}
}

func TestUpdateBalance_Ok(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	body := `{"Currency":"EUR","Amount":79.99}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.AddToBalance(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("AddToBalance expected status %d, got %d", http.StatusOK, rr.Code)
	}

	h.GetAccountByIban(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GetAccountByIban expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var account models.Account
	if err := json.Unmarshal(rr.Body.Bytes(), &account); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if account.IBAN != "EE120000012345678901" {
		t.Fatalf("expected iban EE120000012345678901, got %s", account.IBAN)
	}

	if len(account.Balances) != 1 {
		t.Fatalf("expected one balance, got %d", len(account.Balances))
	}

	aNew, _ := decimal.Parse("179.99")
	if account.Balances[0].Amount != aNew {
		t.Fatalf("EUR balance has wrong amount, got %v", account.Balances[0].Amount)
	}
}

func TestDebitFromBalance_Ok(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	body := `{"Currency":"EUR","Amount":79.99}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DebitFromBalance(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("DebitFromBalance expected status %d, got %d", http.StatusOK, rr.Code)
	}

	h.GetAccountByIban(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GetAccountByIban expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var account models.Account
	if err := json.Unmarshal(rr.Body.Bytes(), &account); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if account.IBAN != "EE120000012345678901" {
		t.Fatalf("expected iban EE120000012345678901, got %s", account.IBAN)
	}

	if len(account.Balances) != 1 {
		t.Fatalf("expected one balance, got %d", len(account.Balances))
	}

	aNew, _ := decimal.Parse("20.01")
	if account.Balances[0].Amount != aNew {
		t.Fatalf("EUR balance has wrong amount, got %v", account.Balances[0].Amount)
	}
}

func TestExchangeCurrency_Ok(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901/EUR/USD/10", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	rctx.URLParams.Add("from", "EUR")
	rctx.URLParams.Add("to", "USD")
	rctx.URLParams.Add("amount", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ExchangeCurrency(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ExchangeCurrency expected status %d, got %d", http.StatusOK, rr.Code)
	}

	h.GetAccountByIban(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GetAccountByIban expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var account models.Account
	if err := json.Unmarshal(rr.Body.Bytes(), &account); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if account.IBAN != "EE120000012345678901" {
		t.Fatalf("expected iban EE120000012345678901, got %s", account.IBAN)
	}

	if len(account.Balances) != 2 {
		t.Fatalf("expected two balance, got %d", len(account.Balances))
	}

	aNew, _ := decimal.Parse("90")
	if account.Balances[0].Amount != aNew {
		t.Fatalf("EUR balance has wrong amount, got %v", account.Balances[0].Amount)
	}

	bNew, _ := decimal.Parse("17.0")
	if account.Balances[1].Amount != bNew {
		t.Fatalf("USD balance has wrong amount, got %v", account.Balances[1].Amount)
	}
}

func TestAddBalance_ZeroAmountFail(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	body := `{"Currency":"EUR","Amount":0}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.AddToBalance(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("AddToBalance expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if rr.Body.String() != "Amount can not be empty, 0 or negative\n" {
		t.Fatalf("AddToBalance expected error message 'Amount can not be empty, 0 or negative', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func TestAddBalance_WrongCurrencyFail(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	body := `{"Currency":"GBP","Amount":100}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.AddToBalance(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("AddToBalance expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if rr.Body.String() != "Currency not found\n" {
		t.Fatalf("AddToBalance expected error message 'Currency not found', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func TestAddBalance_AccountNotFoundFail(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	body := `{"Currency":"EUR","Amount":100}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678903", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678903")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.AddToBalance(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("AddToBalance expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
	const s = "404 page not found\nAccount not found\n"
	if rr.Body.String() != s {
		t.Fatalf("AddToBalance expected error message %q, got %q", s, rr.Body.String())
	}
}

func TestDebitFromBalance_NotEnoughFundsError(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	body := `{"Currency":"EUR","Amount":179.99}`

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DebitFromBalance(rr, req)
	fmt.Println(rr.Body.String())
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("DebitFromBalance expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if rr.Body.String() != "Account does not have enough currency on balance\n" {
		t.Fatalf("DebitFromBalance expected error message 'Account does not have enough currency on balance', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func TestExchangeCurrency_NotEnoughFundsError(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901/EUR/USD/1000", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	rctx.URLParams.Add("from", "EUR")
	rctx.URLParams.Add("to", "USD")
	rctx.URLParams.Add("amount", "1000")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ExchangeCurrency(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ExchangeCurrency expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if rr.Body.String() != "Account does not have enough currency on EUR balance\n" {
		t.Fatalf("ExchangeCurrency expected error message 'Account does not have enough currency on EUR balance', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func TestExchangeCurrency_FromCurrencyNotFoundError(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901/GPB/USD/1000", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	rctx.URLParams.Add("from", "GPB")
	rctx.URLParams.Add("to", "USD")
	rctx.URLParams.Add("amount", "1000")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ExchangeCurrency(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ExchangeCurrency expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if rr.Body.String() != "Currency GPB not found\n" {
		t.Fatalf("ExchangeCurrency expected error message 'Currency GPB not found', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func TestExchangeCurrency_ToCurrencyNotFoundError(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901/EUR/GBP/1000", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	rctx.URLParams.Add("from", "EUR")
	rctx.URLParams.Add("to", "GBP")
	rctx.URLParams.Add("amount", "1000")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ExchangeCurrency(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ExchangeCurrency expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if rr.Body.String() != "Currency GBP not found\n" {
		t.Fatalf("ExchangeCurrency expected error message 'Currency GBP not found', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func TestExchangeCurrency_AccountNotFoundError(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678903/EUR/USD/10", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678903")
	rctx.URLParams.Add("from", "EUR")
	rctx.URLParams.Add("to", "USD")
	rctx.URLParams.Add("amount", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ExchangeCurrency(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ExchangeCurrency expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	if rr.Body.String() != "404 page not found\nAccount not found\n" {
		t.Fatalf("ExchangeCurrency expected error message '404 page not found\nAccount not found', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func TestExchangeCurrency_FromBalanceNotFoundError(t *testing.T) {
	repo := CreateRepoData()
	h := NewAccountHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/accounts/add/EE120000012345678901/USD/EUR/10", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("iban", "EE120000012345678901")
	rctx.URLParams.Add("from", "USD")
	rctx.URLParams.Add("to", "EUR")
	rctx.URLParams.Add("amount", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ExchangeCurrency(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ExchangeCurrency expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if rr.Body.String() != "Account does not have USD balance\n" {
		t.Fatalf("ExchangeCurrency expected error message 'Account does not have USD balance', got '%s'", rr.Body.String())
	}

	ValidateBalanceNotChanged(repo, t)
}

func CreateRepoData() *fakeAccountRepo {
	a, _ := decimal.Parse("100")
	b, _ := decimal.Parse("1.7")
	repo := &fakeAccountRepo{
		accounts: []models.Account{
			{ID: 1, IBAN: "EE120000012345678901", Name: "Account One", Balances: []models.Balance{
				{ID: 1, Account_id: 1, Amount: a, Currency_id: 1, Currency: models.Currency{ID: 1, CurrencyCode: "EUR"}},
			}},
			{ID: 2, IBAN: "EE120000012345678902", Name: "Account Two", Balances: []models.Balance{}},
		},
		currencys: []models.Currency{
			{ID: 1, CurrencyCode: "EUR"},
			{ID: 2, CurrencyCode: "USD"},
		},
		exchangeRates: []models.ExchangeRates{
			{ID: 1, Fromc: 1, Toc: 2, Rate: b},
		},
	}
	return repo
}

func ValidateBalanceNotChanged(repo *fakeAccountRepo, t *testing.T) {
	h2 := NewAccountHandler(repo)

	req2 := httptest.NewRequest(http.MethodGet, "/accounts/EE120000012345678901", nil)
	rr2 := httptest.NewRecorder()

	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("iban", "EE120000012345678901")
	req2 = req2.WithContext(context.WithValue(req2.Context(), chi.RouteCtxKey, rctx2))

	h2.GetAccountByIban(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("GetAccountByIban expected status %d, got %d", http.StatusOK, rr2.Code)
	}

	var account models.Account

	if err := json.Unmarshal(rr2.Body.Bytes(), &account); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if len(account.Balances) != 1 {
		t.Fatalf("expected one balance, got %d", len(account.Balances))
	}

	aNew, _ := decimal.Parse("100")
	if account.Balances[0].Amount != aNew {
		t.Fatalf("EUR balance has wrong amount, got %v", account.Balances[0].Amount)
	}
}

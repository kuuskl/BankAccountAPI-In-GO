package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/govalues/decimal"

	"go-mysql-api/internal/models"
)

type AccountRepository interface {
	Accounts(ctx context.Context) ([]models.Account, error)
	GetAccountByIban(ctx context.Context, iban string) (models.Account, error)
	UpdateBalance(ctx context.Context, balance models.Balance, amount decimal.Decimal) error
	ExchangeBalances(ctx context.Context, account models.Account, fromBalance models.Balance, fromAmount decimal.Decimal, toBalance *models.Balance, toCurrency models.Currency, toAmount decimal.Decimal) error
	AddBalance(ctx context.Context, account models.Account, currency models.Currency, amount decimal.Decimal) error
	GetExchangeRate(ctx context.Context, fromCurrency int64, toCurrency int64) (models.ExchangeRates, error)
	GetCurrencyByCurrencyCode(ctx context.Context, code string) (models.Currency, error)
	//Balances(ctx context.Context) ([]models.Balance, error)
	//Currencys(ctx context.Context) ([]models.Currency, error)
	//ExchangeRates(ctx context.Context) ([]models.ExchangeRates, error)
}

type AccountHandler struct {
	repo AccountRepository
}

func NewAccountHandler(repo AccountRepository) *AccountHandler {
	return &AccountHandler{repo: repo}
}

func (h *AccountHandler) Accounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.repo.Accounts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(accounts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *AccountHandler) GetAccountByIban(w http.ResponseWriter, r *http.Request) {
	iban := chi.URLParam(r, "iban")

	account, err := h.repo.GetAccountByIban(r.Context(), iban)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(account); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *AccountHandler) AddToBalance(w http.ResponseWriter, r *http.Request) {
	var account models.Account
	var currency models.Currency
	var credirDebit models.CreditDebit
	account, currency, credirDebit, err := h.CreditDebitValdation(w, r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "404 page not found\nAccount not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var newBalance bool = true
	for _, balance := range account.Balances {
		if balance.Currency.CurrencyCode == currency.CurrencyCode {
			var newAmount decimal.Decimal
			newAmount, err = balance.Amount.Add(credirDebit.Amount)
			if err != nil {
				http.Error(w, "Error calculating new amount", http.StatusBadRequest)
				return
			}
			err := h.repo.UpdateBalance(r.Context(), balance, newAmount)
			if err != nil {
				http.Error(w, "Error adding to balance", http.StatusBadRequest)
				return
			}
			newBalance = false
		}
	}

	if newBalance {
		err := h.repo.AddBalance(r.Context(), account, currency, credirDebit.Amount)
		if err != nil {
			http.Error(w, "Error adding balance", http.StatusBadRequest)
			return
		}
	}

}

func (h *AccountHandler) DebitFromBalance(w http.ResponseWriter, r *http.Request) {
	var account models.Account
	var currency models.Currency
	var credirDebit models.CreditDebit
	account, currency, credirDebit, err := h.CreditDebitValdation(w, r)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "404 page not found\nAccount not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var newBalance bool = true
	for _, balance := range account.Balances {
		if balance.Currency.CurrencyCode == currency.CurrencyCode {
			var newAmount decimal.Decimal
			newAmount, err = balance.Amount.Sub(credirDebit.Amount)
			if err != nil {
				http.Error(w, "Error calculating new amount", http.StatusBadRequest)
				return
			}

			if newAmount.Cmp(decimal.Decimal{}) < 0 {
				http.Error(w, "Account does not have enough currency on balance", http.StatusBadRequest)
				return
			}

			err := h.repo.UpdateBalance(r.Context(), balance, newAmount)
			if err != nil {
				http.Error(w, "Error debiting from balance", http.StatusBadRequest)
				return
			}
			newBalance = false
		}
	}

	if newBalance {
		http.Error(w, "Account does not have "+currency.CurrencyCode+" balance", http.StatusBadRequest)
		return
	}

}

func (h *AccountHandler) ExchangeCurrency(w http.ResponseWriter, r *http.Request) {
	iban := chi.URLParam(r, "iban")
	from := chi.URLParam(r, "from")
	to := chi.URLParam(r, "to")
	amount, err := decimal.Parse(chi.URLParam(r, "amount"))

	if err != nil {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	if amount.Cmp(decimal.Decimal{}) <= 0 {
		http.Error(w, "Amount can not be empty, 0 or negative", http.StatusBadRequest)
		return
	}

	fromCurrency, err := h.repo.GetCurrencyByCurrencyCode(r.Context(), from)
	if err != nil {
		http.Error(w, "Currency "+from+" not found", http.StatusBadRequest)
		return
	}

	toCurrency, err := h.repo.GetCurrencyByCurrencyCode(r.Context(), to)
	if err != nil {
		http.Error(w, "Currency "+to+" not found", http.StatusBadRequest)
		return
	}

	account, err := h.repo.GetAccountByIban(r.Context(), iban)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "404 page not found\nAccount not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	convertedAmount, err := h.ConvertFromTo(w, r, fromCurrency, toCurrency, amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var fromBalance *models.Balance
	for i := range account.Balances {
		if account.Balances[i].Currency.CurrencyCode == fromCurrency.CurrencyCode {
			fromBalance = &account.Balances[i]
			break
		}
	}

	if fromBalance == nil {
		http.Error(w, "Account does not have "+from+" balance", http.StatusBadRequest)
		return
	}

	newFromAmount, err := fromBalance.Amount.Sub(amount)
	if err != nil {
		http.Error(w, "Error calculating new amount", http.StatusBadRequest)
		return
	}
	if newFromAmount.Cmp(decimal.Decimal{}) <= 0 {
		http.Error(w, "Account does not have enough currency on "+from+" balance", http.StatusBadRequest)
		return
	}

	var toBalance *models.Balance
	var newToAmount = convertedAmount
	for i := range account.Balances {
		if account.Balances[i].Currency.CurrencyCode == toCurrency.CurrencyCode {
			toBalance = &account.Balances[i]
			newToAmount, err = account.Balances[i].Amount.Add(convertedAmount)
			if err != nil {
				http.Error(w, "Error calculating new amount", http.StatusBadRequest)
				return
			}
			break
		}
	}

	if err := h.repo.ExchangeBalances(r.Context(), account, *fromBalance, newFromAmount, toBalance, toCurrency, newToAmount); err != nil {
		http.Error(w, "Error exchanging currency", http.StatusBadRequest)
		return
	}
}

func (h *AccountHandler) contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*60)
}

/* Internal methods */

func (h *AccountHandler) CreditDebitValdation(w http.ResponseWriter, r *http.Request) (account models.Account, currency models.Currency, credirDebit models.CreditDebit, error error) {
	iban := chi.URLParam(r, "iban")
	if err := json.NewDecoder(r.Body).Decode(&credirDebit); err != nil {
		return models.Account{}, models.Currency{}, models.CreditDebit{}, errors.New("Invalid request body")
	}

	if credirDebit.Amount.Cmp(decimal.Decimal{}) <= 0 {
		return models.Account{}, models.Currency{}, models.CreditDebit{}, errors.New("Amount can not be empty, 0 or negative")
	}

	currency, err := h.repo.GetCurrencyByCurrencyCode(r.Context(), credirDebit.Currency)
	if err != nil {
		return models.Account{}, models.Currency{}, models.CreditDebit{}, errors.New("Currency not found")
	}

	account, err = h.repo.GetAccountByIban(r.Context(), iban)
	if err != nil {
		return models.Account{}, models.Currency{}, models.CreditDebit{}, err
	}

	return account, currency, credirDebit, nil

}

func (h *AccountHandler) ConvertFromTo(w http.ResponseWriter, r *http.Request, fromCurrency models.Currency, toCurrency models.Currency, fromAmount decimal.Decimal) (toAmount decimal.Decimal, error error) {
	var rate models.ExchangeRates
	var toAmountReturn decimal.Decimal
	rate, err := h.repo.GetExchangeRate(r.Context(), fromCurrency.ID, toCurrency.ID)
	if err != nil {
		rate, err = h.repo.GetExchangeRate(r.Context(), toCurrency.ID, fromCurrency.ID)
		if err != nil {
			return decimal.Decimal{}, err
		}
		toAmountReturn, err = fromAmount.Quo(rate.Rate)
		if err != nil {
			return decimal.Decimal{}, err
		}
		return toAmountReturn, nil
	}
	toAmountReturn, err = fromAmount.Mul(rate.Rate)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return toAmountReturn, nil
}

/*  For data testing */
/*func (h *AccountHandler) Balances(w http.ResponseWriter, r *http.Request) {
	a, err := h.repo.Balances(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *AccountHandler) Currencys(w http.ResponseWriter, r *http.Request) {
	a, err := h.repo.Currencys(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}*/

/*func (h *AccountHandler) ExchangeRates(w http.ResponseWriter, r *http.Request) {
	a, err := h.repo.ExchangeRates(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}*/

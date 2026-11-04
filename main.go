package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"go-mysql-api/internal/db"
	"go-mysql-api/internal/handlers"
	"go-mysql-api/internal/repository"
)

func main() {
	conn, err := db.Open("root:root@tcp(localhost:3306)/appdb?parseTime=true")
	if err != nil {
		log.Fatal("database connection failed:", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn); err != nil {
		log.Fatal("migration failed:", err)
	}

	repo := repository.NewAccountRepository(conn)
	handler := handlers.NewAccountHandler(repo)

	r := chi.NewRouter()

	r.Route("/accounts", func(r chi.Router) {
		r.Get("/", handler.Accounts)
		r.Get("/{iban}", handler.GetAccountByIban)
		r.Put("/add/{iban}", handler.AddToBalance)
		r.Put("/debit/{iban}", handler.DebitFromBalance)
		r.Put("/exchange/{iban}/{from}/{to}/{amount}", handler.ExchangeCurrency)

		//for data testing
		//r.Get("/balances", handler.Balances)
		//r.Get("/currencys", handler.Currencys)
		//r.Get("/rates", handler.ExchangeRates)
	})

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

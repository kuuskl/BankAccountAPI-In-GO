package db

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var deleteAllData = false // set true to reset database data on each run, useful for testing
var alterTables = false

func Open(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return conn, nil
}

func Migrate(conn *sql.DB) error {
	err := CreateDBAndData(conn)
	if err != nil {
		log.Println("Migrate error:", err)
		return err
	}

	return nil
}

func CreateDBAndData(conn *sql.DB) error {

	err := CreateTables(conn)
	if err != nil {
		log.Println("Failed to create tables:", err)
		return err
	}

	err = InsertAccounts(conn)
	if err != nil {
		log.Println("Failed to insert accounts:", err)
		return err
	}

	err = InsertCurrencys(conn)
	if err != nil {
		log.Println("Failed to insert currencys:", err)
		return err
	}

	err = InsertBalances(conn)
	if err != nil {
		log.Println("Failed to insert balances:", err)
		return err
	}

	err = InsertExchangeRates(conn)
	if err != nil {
		log.Println("Failed to insert exchange rates:", err)
		return err
	}

	return nil
}

func CreateTables(conn *sql.DB) error {
	query := `
			CREATE TABLE IF NOT EXISTS account (
				id BIGINT NOT NULL AUTO_INCREMENT,
				iban VARCHAR(20) NOT NULL,
				name VARCHAR(255) NOT NULL,
				PRIMARY KEY (id)
			)
		`
	_, err := conn.Exec(query)
	if err != nil {
		log.Println("Failed to create account table:", err)
		return err
	}

	query = `
			CREATE TABLE IF NOT EXISTS currency (
				id BIGINT NOT NULL AUTO_INCREMENT,
				currency_code VARCHAR(3) NOT NULL,
				PRIMARY KEY (id)
			)
		`
	_, err = conn.Exec(query)
	if err != nil {
		log.Println("Failed to create currency table:", err)
		return err
	}

	query = `
			CREATE TABLE IF NOT EXISTS balance (
				id BIGINT NOT NULL AUTO_INCREMENT,
				account_id BIGINT NOT NULL,
				amount DECIMAL(19,8) NOT NULL,
				currency_id BIGINT NOT NULL,
				PRIMARY KEY (id)
			)
		`
	_, err = conn.Exec(query)
	if err != nil {
		log.Println("Failed to create balance table:", err)
		return err
	}

	query = `
			CREATE TABLE IF NOT EXISTS exchange_rates (
				id BIGINT NOT NULL AUTO_INCREMENT,
				fromc_id BIGINT NOT NULL,
				toc_id BIGINT NOT NULL,
				rate DECIMAL(19,8) NOT NULL,
				PRIMARY KEY (id)
			)
		`
	_, err = conn.Exec(query)
	if err != nil {
		log.Println("Failed to create exchange_rates table:", err)
		return err
	}

	if deleteAllData {
		err = DeleteAllData(conn)
		if err != nil {
			log.Println("Failed to delete all data:", err)
			return err
		}
	}
	if alterTables {
		AlterTables(conn)
		if err != nil {
			log.Println("Failed to alter tables:", err)
			return err
		}
	}

	return nil

}

func InsertAccounts(conn *sql.DB) error {
	rows, err := conn.Query(`SELECT count(*) FROM account`)

	var cnt int
	for rows.Next() {
		err2 := rows.Scan(&cnt)
		if err2 != nil {
			log.Println("err2:", err2)
			return err
		}
	}

	if rows.Err() != nil {
		log.Println("Failed to read account count:", rows.Err())
		return rows.Err()
	}

	if err != nil || cnt == 0 {
		iban := "EE120000012345678901"
		name := "AccountOne"
		_, err := conn.Exec(`INSERT INTO account (iban, name) VALUES (?, ?)`, iban, name)
		if err != nil {
			log.Println("Failed to insert account one:", err)
			return err
		}

		iban = "EE120000012345678902"
		name = "AccountTwo"
		_, err = conn.Exec(`INSERT INTO account (iban, name) VALUES (?, ?)`, iban, name)
		if err != nil {
			log.Println("Failed to insert account two:", err)
			return err
		}
	}
	defer rows.Close()
	return nil
}

func InsertCurrencys(conn *sql.DB) error {
	rows, err := conn.Query(`SELECT count(*) FROM currency`)

	var cnt int
	for rows.Next() {
		err2 := rows.Scan(&cnt)
		if err2 != nil {
			log.Println("err2:", err2)
			return err
		}
	}

	if rows.Err() != nil {
		log.Println("Failed to read currency count:", rows.Err())
		return rows.Err()
	}

	if err != nil || cnt == 0 {
		_, err := conn.Exec(`INSERT INTO currency (currency_code) VALUES (?)`, "EUR")
		if err != nil {
			log.Println("Failed to insert EUR currency:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO currency (currency_code) VALUES (?)`, "USD")
		if err != nil {
			log.Println("Failed to insert USD currency:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO currency (currency_code) VALUES (?)`, "GBP")
		if err != nil {
			log.Println("Failed to insert GBP currency:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO currency (currency_code) VALUES (?)`, "SEK")
		if err != nil {
			log.Println("Failed to insert SEK currency:", err)
			return err
		}
	}
	defer rows.Close()

	return nil
}

func InsertBalances(conn *sql.DB) error {
	rows, err := conn.Query(`SELECT count(*) FROM balance`)

	var cnt int
	for rows.Next() {
		err2 := rows.Scan(&cnt)
		if err2 != nil {
			log.Println("err2:", err2)
			return err
		}
	}
	if rows.Err() != nil {
		log.Println("Failed to read balance count:", rows.Err())
		return rows.Err()
	}
	if err != nil || cnt == 0 {

		var (
			id       int
			account  int
			amount   string
			currency int
			eur_id   int
			gpb_id   int
			acc_id   int
		)

		query_c := `SELECT id FROM currency WHERE currency_code = ?`
		err = conn.QueryRow(query_c, "EUR").Scan(&eur_id)
		if err != nil {
			log.Println("EUR currency not found :", err)
			return err
		}

		query_c = `SELECT id FROM currency WHERE currency_code = ?`
		err = conn.QueryRow(query_c, "GBP").Scan(&gpb_id)
		if err != nil {
			log.Println("EUR currency not found :", err)
			return err
		}

		query_a := `SELECT id FROM account WHERE iban = ?`
		err = conn.QueryRow(query_a, "EE120000012345678901").Scan(&acc_id)
		if err != nil {
			log.Println("Account with iban EE120000012345678901 not found:", err)
			return err
		}

		// Query the database and scan the values into out variables.
		query := `SELECT id, account_id, amount, currency_id FROM balance WHERE id = (SELECT id FROM account WHERE iban = ?) AND currency_id = ?`
		err = conn.QueryRow(query, "EE120000012345678901", eur_id).Scan(&id, &account, &amount, &currency)

		//no balances, add balance
		if err != nil {
			_, err := conn.Exec(`INSERT INTO balance (account_id, amount, currency_id) VALUES (?, ?, ?)`, acc_id, 100, eur_id)
			if err != nil {
				log.Println("Failed to insert EUR currency:", err)
				return err
			}

			_, err = conn.Exec(`INSERT INTO balance (account_id, amount, currency_id) VALUES (?, ?, ?)`, acc_id, 200, gpb_id)
			if err != nil {
				log.Println("Failed to insert EUR currency:", err)
				return err
			}

		}
	}
	defer rows.Close()

	return nil
}

func InsertExchangeRates(conn *sql.DB) error {
	rows, err := conn.Query(`SELECT count(*) FROM exchange_rates`)

	var cnt int
	for rows.Next() {
		err2 := rows.Scan(&cnt)
		if err2 != nil {
			log.Println("err2:", err2)
			return err
		}
	}

	if rows.Err() != nil {
		log.Println("Failed to read exchange rate count:", rows.Err())
		return rows.Err()
	}

	if err != nil || cnt == 0 {

		rows, err := conn.Query(`SELECT id, currency_code FROM currency`)

		var eur_id int
		var usd_id int
		var gbp_id int
		var sek_id int
		var id int
		var code string
		for rows.Next() {
			err2 := rows.Scan(&id, &code)
			if err2 != nil {
				log.Println("err2:", err2)
				return err
			}
			switch code {
			case "EUR":
				eur_id = id
			case "USD":
				usd_id = id
			case "GBP":
				gbp_id = id
			case "SEK":
				sek_id = id
			}
		}
		if err := rows.Err(); err != nil {
			log.Println("Failed to read currencies:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO exchange_rates (fromc_id, toc_id, rate) VALUES (?, ?, ?)`, eur_id, usd_id, 1.7)
		if err != nil {
			log.Println("Failed to insert EUR / USD rate:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO exchange_rates (fromc_id, toc_id, rate) VALUES (?, ?, ?)`, eur_id, gbp_id, 0.87)
		if err != nil {
			log.Println("Failed to insert EUR / GBP rate:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO exchange_rates (fromc_id, toc_id, rate) VALUES (?, ?, ?)`, eur_id, sek_id, 1.70)
		if err != nil {
			log.Println("Failed to insert EUR / SEK rate:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO exchange_rates (fromc_id, toc_id, rate) VALUES (?, ?, ?)`, usd_id, gbp_id, 0.75)
		if err != nil {
			log.Println("Failed to insert USD / GPB rate:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO exchange_rates (fromc_id, toc_id, rate) VALUES (?, ?, ?)`, usd_id, sek_id, 9.14)
		if err != nil {
			log.Println("Failed to insert USD / SEK rate:", err)
			return err
		}

		_, err = conn.Exec(`INSERT INTO exchange_rates (fromc_id, toc_id, rate) VALUES (?, ?, ?)`, gbp_id, sek_id, 12.24)
		if err != nil {
			log.Println("Failed to insert GPB / SEK rate:", err)
			return err
		}

	}
	defer rows.Close()

	return nil
}

/* For testing */

func DeleteAllData(conn *sql.DB) error {

	_, err := conn.Exec(`DELETE FROM account`)
	if err != nil {
		log.Println("Failed delete:", err)
		return err
	}

	_, err = conn.Exec(`DELETE FROM currency`)
	if err != nil {
		log.Println("Failed delete:", err)
		return err
	}

	_, err = conn.Exec(`DELETE FROM balance`)
	if err != nil {
		log.Println("Failed delete:", err)
		return err
	}

	_, err = conn.Exec(`DELETE FROM exchange_rates`)
	if err != nil {
		log.Println("Failed delete:", err)
		return err
	}

	return nil

}

func AlterTables(conn *sql.DB) error {

	_, err := conn.Exec(`ALTER TABLE exchange_rates MODIFY rate DECIMAL(19,8)`)
	if err != nil {
		log.Println("Alter table exchange_rates failed:", err)
		return err
	}

	_, err = conn.Exec(`ALTER TABLE balance MODIFY amount DECIMAL(19,8)`)
	if err != nil {
		log.Println("Alter table balance failed:", err)
		return err
	}

	return nil

}

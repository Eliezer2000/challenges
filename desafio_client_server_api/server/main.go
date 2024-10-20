package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	_ "github.com/go-sql-driver/mysql"
)

type Cotacao struct {
	Bid string `json:"bid"`
}

func main() {

	dsn := "root:admin@tcp(localhost:3306)/goexpert"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal("Erro ao conectar ao banco de dados:", err)
    }
    defer db.Close()

    createTable := `CREATE TABLE IF NOT EXISTS cotacoes (
                        id INT AUTO_INCREMENT PRIMARY KEY,
                        bid VARCHAR(10),
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                    )`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
		defer cancel()

		cotacao, err := fetchCotacao(ctx)
		if err != nil {
			http.Error(w, "Timeout na API externa", http.StatusRequestTimeout)
			log.Println("Error ao buscar cotacao:", err)
			return
		}

		ctxDB, cancelDB := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancelDB()

		err = saveCotacao(ctxDB, db, cotacao.Bid)
		if err != nil {
			http.Error(w, "Erro ao salvar cotacao", http.StatusInternalServerError)
			log.Println("Error ao salvar a cotacao no banco:", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cotacao)
	})

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func fetchCotacao(ctx context.Context) (Cotacao, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return Cotacao{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Cotacao{}, err
	}
	defer resp.Body.Close()

	var result map[string]Cotacao
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Cotacao{}, err
	}

	return result["USDBRL"], nil
}

func saveCotacao(ctx context.Context, db *sql.DB, bid string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, err := db.ExecContext(ctx, "INSERT INTO cotacoes (bid) VALUES (?)", bid)
		return err
	}
}
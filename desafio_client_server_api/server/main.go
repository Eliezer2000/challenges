package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
    _ "github.com/mattn/go-sqlite3"
)

// Estrutura para mapear a resposta da API externa
type APIResponse struct {
    USDBRL struct {
        Bid string `json:"bid"`
    } `json:"USDBRL"`
}

// Estrutura para a resposta do servidor ao cliente 
type ServerResponse struct {
    Bid string `json:"bid"`
}

func main() {
    // Inicializando o banco de dados SQLite
    db, err := sql.Open("sqlite3", "./cotacao.db")
    if err != nil {
        log.Fatalf("Erro ao abrir o banco de dados: %v", err)
    }
    defer db.Close()

    // Criar a tabela se nao existir
    createTableSQL := `CREATE TABLE IF NOT EXISTS cotacoes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        bid TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

    _, err = db.Exec(createTableSQL)
    if err != nil {
        log.Fatalf("Erro ao criar tabela: %v", err)
    }

    // Define o handler para o endpoit /cotacao
    http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
        // Contexto com timeout de 200ms para a chamada da API externa
        ctxAPI, cancelAPI := context.WithTimeout(context.Background(), 200 * time.Millisecond)
        defer cancelAPI()

        // Requisicao da API externa
        req, err := http.NewRequestWithContext(ctxAPI, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
        if err != nil {
            http.Error(w, "Erro ao criar requisição para a API externa", http.StatusInternalServerError)
            log.Printf("Erro ao criar requisição: %v", err)
            return
        }

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            http.Error(w, "Erro ao buscar cotação na API externa", http.StatusInternalServerError)
            log.Printf("Erro ao buscar cotação: %v", err)
            return
        }
        defer resp.Body.Close()

        // Decodifica a resposta da API externa
        var apiResp APIResponse
        if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
            http.Error(w, "Erro ao decodificar resposta da API externa", http.StatusInternalServerError)
            log.Printf("Erro ao decodificar JSON: %v", err)
            return
        }
        
        bid := apiResp.USDBRL.Bid

        // Contexto com timeout de 10ms para inserir no banco de dados 
        ctxDB, cancelDB := context.WithTimeout(context.Background(), 10 * time.Millisecond)
        defer cancelDB()

        // Insere a cotacao no banco de dados
        insertSQL := `INSERT INTO cotacoes (bid) VALUES (?)`
        _, err = db.ExecContext(ctxDB, insertSQL, bid)
        if err != nil {
            http.Error(w, "Erro ao salvar cotacoes no banco de dados", http.StatusInternalServerError)
            log.Printf("Erro ao inserir no DB: %v", err)
            return
        }

        // Prepara a resposta ao cliente 
        serverResp := ServerResponse{
            Bid: bid,
        }

        w.Header().Set("Content-Type", "application/json")
        if err := json.NewEncoder(w).Encode(serverResp); err != nil {
            http.Error(w, "Erro ao encodar resposta JSON", http.StatusInternalServerError)
            log.Printf("Erro ao encodar JSON: %v", err)
            return
        }
    })
    log.Println("Servidor iniciado na porta 8080...")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatalf("Erro ao iniciar o servidor: %v", err)
    }
}
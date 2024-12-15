package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Estrutura para mapear a resposta do servidor
type ServerResponse struct {
    Bid string `json:"bid"`
}

func main() {
    // Contexto com timeout de 300ms para a requisicao ao servidor
    ctx, cancel := context.WithTimeout(context.Background(), 300 * time.Millisecond)
    defer cancel()

    // Criar a requisicao HTTP com o contexto
    req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
    if err != nil {
        log.Fatalf("Erro ao criar a requisição: %v", err)
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("Erro ao fazer a requisiçao: %v", err)
    }
    defer resp.Body.Close()

    // Decodifica a resposta JSON
    var serverResp ServerResponse
    if err := json.NewDecoder(resp.Body).Decode(&serverResp); err != nil {
        log.Fatalf("Erro ao decodificar resposta JSON: %v", err)
    }

    // formata o conteúdo para salvar no arquivo
    content := "Dólar" + serverResp.Bid + "\n"

    // Escreve no arquivo "cotacao.txt"
    err = ioutil.WriteFile("cotacao.txt", []byte(content), 0644)
    if err != nil {
        log.Fatalf("Erro ao escrever no arquivo: %v", err)
    }

    log.Println("Cotacao salva com sucesso em cotacao.txt")
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Cotacao struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		log.Fatal("Erro ao criar a requisicao: ", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("Erro ao fazer requisicao:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Erro na resposta do servidor: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Erro ao ler o corpo da resposta:", err)
	}

	log.Println("Resposta do servidor:", string(body))

	var cotacao Cotacao
	if err := json.Unmarshal(body, &cotacao); err != nil {
		log.Fatal("Erro ao decodificar resposta:", err)
	}

	if cotacao.Bid == "" {
		log.Fatal("Erro: cotação 'bid' não foi preenchida corretamente")
	}

	err = writeToFile(cotacao.Bid)
	if err != nil {
		log.Fatal("Erro ao escrever no arquivo:", err)
	}
}

func writeToFile(bid string) error {
	content := fmt.Sprintf("Dólar: %s", bid)
	return os.WriteFile("cotacao.txt", []byte(content), os.ModePerm)
}
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Adress struct {
	CEP          string `json:"cep"`
	Street       string `json:"street"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	State        string `json:"state"`
	API          string `json:"-"`
}
/*
Modelo unificado para endereço, compatível com ambas APIs
API armazena a origem dos dados (não é serializado em JSON)
*/


type CepResult struct {
	API   string
	Data  *Adress
	Error error
}
/*
Canal de comunicação entre goroutines
Transporta resultado ou erro + identificação da API
*/



type BrasilApiResponse struct {
	CEP          string `json:"cep"`
	State        string `json:"state"`
	City         string `json:"city"`
	Neighborhood string `json:"neighborhood"`
	Street       string `json:"street"`
}

type ViaCEPResponse struct {
	CEP        string `json:"cep"`
	Logradouro string `json:"logradouro"`
	Bairro     string `json:"bairro"`
	Localidade string `json:"localidade"`
	UF         string `json:"uf"`
	Erro       bool   `json:"erro"`
}
/*
Estruturas específicas para cada API
Mapeiam campos diferentes das respostas
*/



func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <CEP>")
		return
	}
	cep := os.Args[1]
/*
Valida argumento da linha de comando
Extrai CEP fornecido
*/

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 1)
	defer cancel()
/*
Cria contexto com timeout de 1 segundo
defer cancel() garante liberação de recursos
*/


	resultChan := make(chan CepResult, 2)
/*
Canal bufferizado para 2 resultados (uma por API) 
*/


	go fetchBrasilAPI(ctx, cep, resultChan)
	go fetchViaCEP(ctx, cep, resultChan)
/*
Inicia 2 goroutines para buscar nas APIs paralelamente
Passa contexto, CEP e canal de resultados
*/


	select {
	case res := <-resultChan:
		cancel()
		if res.Error != nil {
			fmt.Printf("Erro da API %s: %v\n", res.API, res.Error)
		} else {
			printAddress(res.Data)
		}
		case <-ctx.Done():
			fmt.Println("Timeout: nenhuma resposta recebida em 1 segundo")
	}
/*
select aguarda o primeiro evento:
	- Primeiro resultado válido no canal
	- Timeout do contexto
Cancela operações pendentes ao receber resposta
Exibe resultado ou erro
*/
}

func fetchBrasilAPI(ctx context.Context, cep string, ch chan<- CepResult) {
	 // 1. Constrói a URL da API com o CEP fornecido
	url := fmt.Sprintf("https://brasilapi.com.br/api/cep/v1/%s", cep)

	// 2. Cria uma nova request HTTP com contexto de timeout/cancelamento
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		 // 3. Se falhar na criação da request, envia erro pelo cnal
		ch <- CepResult{API: "BrasilAPI", Error: err}
		return
	}

	// 4. Executa a requisição HTTP
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// 5. Se falhar na execução, envia erro pelo canal
		ch <- CepResult{API: "BrasilAPI", Error: err}
		return
	}

	 // 6. Garante que o corpo da resposta será fechado
	defer resp.Body.Close()

	 // 7. Verifica se o status code é 200 OK
	if resp.StatusCode != http.StatusOK {
		ch <- CepResult{API: "BrasilAPI", Error: fmt.Errorf("status code %d", resp.StatusCode)}
		return
	}

	// 8. Decodifica o JSON da resposta para a struct específica
	var data BrasilApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		ch <- CepResult{API: "BrasilAPI", Error: err}
		return	
	}

	// 9. Converte a resposta específica para o formato padrão
	ch <- CepResult{
		API:  "BrasilAPI",
		Data: &Adress{
			CEP:          data.CEP,
			Street:       data.Street,
			Neighborhood: data.Neighborhood,
			City:         data.City,
			State:        data.State,
			API: 		"BrasilAPI",
		},
	}
}

func fetchViaCEP(ctx context.Context, cep string, ch chan<- CepResult) {
	// 1. Constrói a URL da API com o CEP
	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json", cep)

	// 2. Cria request com contexto
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		ch <- CepResult{API: "ViaCEP", Error: err}
		return
	}

	// 3. Executa a requisição
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ch <- CepResult{API: "ViaCEP", Error: err}
		return
	}

	// 4. Garante fechamento do corpo
	defer resp.Body.Close()

	// 5. Verifica status code
	if resp.StatusCode != http.StatusOK {
		ch <- CepResult{API: "ViaCEP", Error: fmt.Errorf("status code %d", resp.StatusCode)}
		return
	}

	// 6. Decodifica JSON para struct específica
	var data ViaCEPResponse	
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		ch <- CepResult{API: "ViaCEP", Error: err}
		return
	}

	// 7. Verifica flag de erro especifica da ViaCEp
	if data.Erro {
		ch <- CepResult{API: "ViaCEP", Error: errors.New("CEP não encontrado")}
		return
	}

	// 8. Converte camps com nomes diferentes
	ch <- CepResult{
		API:  "ViaCEP",
		Data: &Adress{
			CEP:          data.CEP,
			Street:       data.Logradouro, // Mapeia "Logradouro" para "Street"
			Neighborhood: data.Bairro,     // Mapeia "Bairro" para "Neighborhood"
			City:         data.Localidade, // Mapeia "Localidade" para "City"
			State:        data.UF,         // Mapeia "UF" para "State"
			API: 		"ViaCEP",
		},
	}
}

func printAddress(addr *Adress) {
	fmt.Printf("API: %s\n", addr.API)
	fmt.Printf("CEP: %s\n", addr.CEP)
	fmt.Printf("Rua: %s\n", addr.Street)
	fmt.Printf("Bairro: %s\n", addr.Neighborhood)
	fmt.Printf("Cidade: %s\n", addr.City)
	fmt.Printf("Estado: %s\n", addr.State)
}
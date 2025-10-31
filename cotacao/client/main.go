package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Estrutura para parse da resposta do servidor
type CotacaoResponse struct {
	Bid string `json:"bid"`
}

func main() {
	// Contexto com timeout de 300ms
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Chamada para o servidor
	bid, err := buscarCotacaoServidor(ctx)
	if err != nil {
		log.Printf("Erro ao buscar cotação: %v", err)
		return
	}

	// Salva cotação no arquivo
	if err := salvarCotacaoArquivo(bid); err != nil {
		log.Printf("Erro ao salvar cotação em arquivo: %v", err)
		return
	}

	fmt.Printf("Cotação do dólar salva: %s\n", bid)
}

// Obter cotação do servidor
func buscarCotacaoServidor(ctx context.Context) (string, error) {
	// Criando requisição do contexto
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição: %w", err)
	}

	// Realizando requisição
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Verifica se ocorreu timeout do contexto
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout de 300ms exedido na requisição ao servidor")
		}
		return "", fmt.Errorf("erro na requisição HTTP: %w", err)
	}
	defer resp.Body.Close()

	// Status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("servidor retornou status: %d", resp.StatusCode)
	}

	// Deserializa resposta JSON
	var cotacao CotacaoResponse
	if err := json.NewDecoder(resp.Body).Decode(&cotacao); err != nil {
		return "", fmt.Errorf("erro ao deserializar JSON: %w", err)
	}

	if cotacao.Bid == "" {
		return "", fmt.Errorf("campo bid está vázio na resposta")
	}

	return cotacao.Bid, nil
}

// Salva cotação no arquivo cotacao.txt

func salvarCotacaoArquivo(bid string) error {
	// Conteúdo
	conteudo := fmt.Sprintf("Dólar: %s\n", bid)

	// Cria, escreve ou sobrescreve em arquivo
	err := os.WriteFile("contacao.txt", []byte(conteudo), 0644)
	if err != nil {
		return fmt.Errorf("erro ao escrever no arqivo: %w", err)
	}

	log.Printf("Arquivo cotacao.txt salvo com sucesso")
	return nil
}

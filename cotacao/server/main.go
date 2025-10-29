package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

// Estrutura json
type APIResponse struct {
	USDBRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}

// Estrutura de responsta ao cliente da requisição
type ContacaoResponse struct {
	Bid string `json:"bid"`
}

var db *sql.DB

// Função para inicializar a conexão com SQLite, validando banco de dados e tabela
func init() {
	var err error
	db, err = sql.Open("sqlite", "./cotacao.db")
	if err != nil {
		log.Fatal("Erro ao abrir banco de dados:", err)
	}

	// Verificando conexão ao bd
	if err = db.Ping(); err != nil {
		log.Fatal("Erro ao conectar com banco de dados:", err)
	}

	// Criação da tabela se não existir, sem uso de ORM
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS cotacoes(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		bid TEXT NOT NULL
	);`

	// Contexto com limite de 5 segundos
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = db.ExecContext(ctx, createTableSQL)
	if err != nil {
		log.Fatal("Erro ao criar tabela:", err)
	}

	log.Println("Banco de dados inicializado com sucesso")
}

func main() {
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	http.HandleFunc("/cotacao", handleCotacao)

	log.Println("Servidor iniciado na porta 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}

func handleCotacao(w http.ResponseWriter, r *http.Request) {
	// Obtem a contação com timeout de 200ms.
	ctxAPI, cancelAPI := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancelAPI()

	bid, err := obterCotacaoAPI(ctxAPI)
	if err != nil {
		log.Printf("Erro ao obter cotação: %v", err)

		// Se ocorrer timeout
		if ctxAPI.Err() == context.DeadlineExceeded {
			http.Error(w, "Timeou na consulta da API externa", http.StatusRequestTimeout)
		} else {
			http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		}
		return
	}

	// Se houver sucesso ao retornar, salva no banco de dados, com time out de 10ms.
	ctxDB, cancelDB := context.WithTimeout(r.Context(), 10*time.Millisecond)
	defer cancelDB()

	if err := salvarCotacaoDB(ctxDB, bid); err != nil {
		// Loga o erro
		log.Printf("Erro ao salvar cotação no banco: %v", err)
	}

	// Retornando resposta ao cliente da requisição
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := ContacaoResponse{Bid: bid}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Erro ao serializar resposta: %v", err)
		http.Error(w, "Erro ao processar resposta", http.StatusInternalServerError)
	}

}

// Método para buscar cotação na API
func obterCotacaoAPI(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro na requisição HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API retornou status %d", resp.StatusCode)
	}

	var APIResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&APIResponse); err != nil {
		return "", fmt.Errorf("erro ao deserializar JSON: %w", err)
	}

	if APIResponse.USDBRL.Bid == "" {
		return "", fmt.Errorf("campo bid está vázio no retorno")
	}

	return APIResponse.USDBRL.Bid, nil

}

// Salva a contação no banco de dados
func salvarCotacaoDB(ctx context.Context, bid string) error {
	query := "INSERT INTO cotacoes(bid) VALUES(?)"
	_, err := db.ExecContext(ctx, query, bid)
	if err != nil {
		return fmt.Errorf("erro ao executar query: %w", err)
	}
	log.Printf("Cotação salva no banco de dados: %s", bid)
	return nil
}

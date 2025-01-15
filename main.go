package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/go-sql-driver/mysql"
)

type model struct {
	userInput  string
	output     string
	user       string
	pass       string
	dbName     string
	db         *sql.DB
	connected  bool
	promptStep int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			input := strings.TrimSpace(m.userInput)
			m.userInput = ""

			switch m.promptStep {
			case 0:
				m.user = input
				m.output = "Digite a senha:"
				m.promptStep++
			case 1:
				m.pass = input
				m.output = "Digite o nome do banco de dados:"
				m.promptStep++
			case 2:
				m.dbName = input
				if err := m.connectToDatabase(); err != nil {
					m.output = fmt.Sprintf("Erro ao conectar ao banco: %v", err)
					return m, nil
				}
				m.output = "Conexão bem-sucedida! Digite comandos SQL ou 'sair' para desconectar."
				m.promptStep++
			case 3:
				if strings.ToLower(input) == "sair" {
					m.disconnectFromDatabase()
					return m, tea.Quit
				}

				m.output = m.executeSQL(input)
			}
		} else {
			m.userInput += msg.String()
		}
	}

	return m, nil
}

func (m *model) connectToDatabase() error {
	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", m.user, m.pass, m.dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return err
	}

	m.db = db
	m.connected = true
	return nil
}

func (m *model) disconnectFromDatabase() {
	if m.db != nil {
		m.db.Close()
		m.db = nil
		m.connected = false
	}
}

func (m *model) executeSQL(query string) string {
	if m.db == nil {
		return "Erro: conexão com o banco não está ativa."
	}

	rows, err := m.db.Query(query)
	if err != nil {
		return fmt.Sprintf("Erro ao executar comando: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Sprintf("Erro ao obter colunas: %v", err)
	}

	var results []string
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return fmt.Sprintf("Erro ao ler resultado: %v", err)
		}

		row := []string{}
		for _, value := range values {
			if b, ok := value.([]byte); ok {
				row = append(row, string(b)) // Converte []byte para string
			} else {
				row = append(row, fmt.Sprintf("%v", value))
			}
		}
		results = append(results, strings.Join(row, "\t"))
	}

	if err := rows.Err(); err != nil {
		return fmt.Sprintf("Erro durante a leitura das linhas: %v", err)
	}

	if len(results) == 0 {
		return "Nenhum resultado encontrado."
	}

	return strings.Join(results, "\n")
}

func (m model) View() string {
	if m.promptStep == 0 {
		return "Digite o usuário do banco de dados:\n> " + m.userInput
	}
	return m.output + "\n> " + m.userInput
}

func main() {
	p := tea.NewProgram(model{})
	if err := p.Start(); err != nil {
		log.Fatalf("Erro ao iniciar a aplicação: %v", err)
	}
}

package database

import (
	"context"
	"time"
)

// Validator provides methods to validate the database state
type Validator struct {
	conn *Connection
}

// NewValidator creates a new database validator
func NewValidator(conn *Connection) *Validator {
	return &Validator{conn: conn}
}

// ValidateConnection checks if the database connection is active
func (v *Validator) ValidateConnection() (bool, string) {
	if v.conn == nil || v.conn.Client == nil {
		return false, "Conexão não inicializada"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := v.conn.Client.Ping(ctx, nil)
	if err != nil {
		return false, "Erro ao conectar ao banco de dados. Verifique se você restaurou alguma base."
	}

	return true, "Conexão com o banco de dados estabelecida com sucesso. \\o/"
}

// IsDatabaseEmpty checks if the database has any stock records
func (v *Validator) IsDatabaseEmpty() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := v.conn.GetCollection(CollectionEstoques).CountDocuments(ctx, map[string]interface{}{})
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

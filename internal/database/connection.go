package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connection holds the MongoDB client and database reference
type Connection struct {
	Client   *mongo.Client
	Database *mongo.Database
}

var instance *Connection

// GetConnection returns the singleton database connection
func GetConnection() *Connection {
	return instance
}

// Connect creates a new MongoDB connection
func Connect() (*Connection, error) {
	// Load .env file
	godotenv.Load()

	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := "12220"
	dbName := "DigisatServer"

	if user == "" || pass == "" || host == "" {
		return nil, fmt.Errorf("variáveis de ambiente DB_USER, DB_PASS e DB_HOST devem estar definidas")
	}

	// URL encode credentials
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/?serverSelectionTimeoutMS=5000",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		url.QueryEscape(host),
		port,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri).SetMaxPoolSize(50)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar ao MongoDB: %w", err)
	}

	// Ping to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar conexão: %w", err)
	}

	instance = &Connection{
		Client:   client,
		Database: client.Database(dbName),
	}

	return instance, nil
}

// Disconnect closes the MongoDB connection
func (c *Connection) Disconnect() error {
	if c.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return c.Client.Disconnect(ctx)
	}
	return nil
}

// IsConnected checks if the database connection is alive
func (c *Connection) IsConnected() bool {
	if c.Client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return c.Client.Ping(ctx, nil) == nil
}

// GetCollection returns a reference to a collection
func (c *Connection) GetCollection(name string) *mongo.Collection {
	return c.Database.Collection(name)
}

// Collection names as constants
const (
	CollectionPessoas                 = "Pessoas"
	CollectionProdutosServicos        = "ProdutosServicos"
	CollectionProdutosServicosEmpresa = "ProdutosServicosEmpresa"
	CollectionEstoques                = "Estoques"
	CollectionTributacoesEstadual     = "TributacoesEstadual"
	CollectionMovimentacoes           = "Movimentacoes"
	CollectionRecebimentos            = "Recebimentos"
	CollectionTurnosLancamentos       = "TurnosLancamentos"
	CollectionConfiguracoesServidor   = "ConfiguracoesServidor"
	CollectionTributacoesFederal      = "TributacoesFederal"
)

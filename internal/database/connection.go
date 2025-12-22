package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"sync"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connection holds the MongoDB client and database reference
type Connection struct {
	Client   *mongo.Client
	Database *mongo.Database
}

var (
	instance *Connection
	once     sync.Once
)

// GetConnection returns the singleton database connection
func GetConnection() *Connection {
	return instance
}

// Connect creates a new MongoDB connection
func Connect() (*Connection, error) {
	var err error

	// Load .env file
	godotenv.Load()

	once.Do(func() {
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASS")
		host := os.Getenv("DB_HOST")

		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "12220"
		}

		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "DigisatServer"
		}

		if user == "" || pass == "" || host == "" {
			err = fmt.Errorf("variáveis de ambiente DB_USER, DB_PASS e DB_HOST devem estar definidas")
			return
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
		var client *mongo.Client
		client, err = mongo.Connect(ctx, clientOptions)
		if err != nil {
			err = fmt.Errorf("erro ao conectar ao MongoDB: %w", err)
			return
		}

		// Ping to verify connection
		err = client.Ping(ctx, nil)
		if err != nil {
			err = fmt.Errorf("erro ao verificar conexão: %w", err)
			return
		}

		instance = &Connection{
			Client:   client,
			Database: client.Database(dbName),
		}
	})

	return instance, err
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
	CollectionPagamentos              = "Pagamentos"
	CollectionAbastecimentos          = "Abastecimentos"
	CollectionConfiguracoes           = "Configuracoes"

	// Additional collections for complete emitente deletion (from legacy)
	CollectionAgendamentos                            = "Agendamentos"
	CollectionAnunciosMercadoLivre                    = "AnunciosMercadoLivre"
	CollectionArquivosDigisatContabil                 = "ArquivosDigisatContabil"
	CollectionBicos                                   = "Bicos"
	CollectionBoletos                                 = "Boletos"
	CollectionBoletosSemParcelaRecebimentoBoleto      = "BoletosSemParcelaRecebimentoBoleto"
	CollectionCartasCorrecao                          = "CartasCorrecao"
	CollectionCheques                                 = "Cheques"
	CollectionComissoes                               = "Comissoes"
	CollectionConferenciasEstoque                     = "ConferenciasEstoque"
	CollectionConhecimentosTransporteEletronico       = "ConhecimentosTransporteEletronico"
	CollectionConsultasServicoCredito                 = "ConsultasServicoCredito"
	CollectionControlesEntrega                        = "ControlesEntrega"
	CollectionDadosDigisatContabil                    = "DadosDigisatContabil"
	CollectionDadosDigisatScanntech                   = "DadosDigisatScanntech"
	CollectionDevolucoes                              = "Devolucoes"
	CollectionEntregasDelivery                        = "EntregasDelivery"
	CollectionEstoquesFisicos                         = "EstoquesFisicos"
	CollectionEstoquesFisicosMovimentacaoInterna      = "EstoquesFisicosMovimentacaoInterna"
	CollectionGestaoContratos                         = "GestaoContratos"
	CollectionInternacoes                             = "Internacoes"
	CollectionInutilizacoes                           = "Inutilizacoes"
	CollectionItensCreditoDebitoCashback              = "ItensCreditoDebitoCashback"
	CollectionItensCreditoDebitoCliente               = "ItensCreditoDebitoCliente"
	CollectionItemMesaContaOcorrencias                = "ItemMesaContaOcorrencias"
	CollectionItensMesaConta                          = "ItensMesaConta"
	CollectionItensPedidoRestaurante                  = "ItensPedidoRestaurante"
	CollectionManifestacaoDestinatario                = "ManifestacaoDestinatario"
	CollectionManifestosEletronicoDocumentoFiscal     = "ManifestosEletronicoDocumentoFiscal"
	CollectionMaosObra                                = "MaosObra"
	CollectionMovimentosConta                         = "MovimentosConta"
	CollectionOrcamentosIndustria                     = "OrcamentosIndustria"
	CollectionOrdensProducao                          = "OrdensProducao"
	CollectionReceitas                                = "Receitas"
	CollectionRomaneiosCarga                          = "RomaneiosCarga"
	CollectionValesPresente                           = "ValesPresente"
	CollectionTurnos                                  = "Turnos"
	CollectionJornadasTrabalho                        = "JornadasTrabalho"
	CollectionFolhasLmc                               = "FolhasLmc"
	CollectionMesasContasClienteBloqueadas            = "MesasContasClienteBloqueadas"
	CollectionDescontinuidadesEncerrante              = "DescontinuidadesEncerrante"
	CollectionSaldosIcmsStRetido                      = "SaldosIcmsStRetido"
	CollectionXmlMovimentacoes                        = "XmlMovimentacoes"
	CollectionOrdensCardapio                          = "OrdensCardapio"
	CollectionRegistrosPafEcf                         = "RegistrosPafEcf"
	CollectionArquivosSngpc                           = "ArquivosSngpc"
	CollectionConhecimentosTransporteRodoviarioCargas = "ConhecimentosTransporteRodoviarioCargas"
)

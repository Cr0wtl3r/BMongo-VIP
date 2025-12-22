package operations

import (
	"context"
	"fmt"
	"time"

	"BMongo-VIP/internal/database"

	"go.mongodb.org/mongo-driver/bson"
)


func (m *Manager) CleanDatabase(log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	collections, err := m.conn.Database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("erro ao listar coleções: %w", err)
	}


	preserve := map[string]bool{
		"system.indexes":             true,
		"system.users":               true,
		"system.version":             true,
		"startup_log":                true,
		"ConfiguracoesServidor":      true,
		"ConfiguracoesSincronizacao": true,
		"DigisatUpdate":              true,
		"Pessoas":                    true,
		"SequenciasDocumentos":       true,
		"Estados":                    true,
		"Cidades":                    true,
	}






	log("Iniciando limpeza da base de dados...")

	for _, colName := range collections {
		if m.state.ShouldStop() {
			log("Operação cancelada.")
			return nil
		}

		if preserve[colName] {
			if colName == database.CollectionPessoas {

				log(fmt.Sprintf("Limpando coleção %s (mantendo Emitentes)...", colName))
				_, err := m.conn.GetCollection(colName).DeleteMany(ctx, bson.M{"_t": bson.M{"$ne": "Emitente"}})
				if err != nil {
					log(fmt.Sprintf("Erro ao limpar %s: %s", colName, err.Error()))
				}
			}
			continue
		}

		log(fmt.Sprintf("Removendo coleção %s...", colName))
		if err := m.conn.GetCollection(colName).Drop(ctx); err != nil {
			log(fmt.Sprintf("Erro ao remover coleção %s: %s", colName, err.Error()))
		}
	}

	log("Base de dados limpa com sucesso!")
	return nil
}


func (m *Manager) CreateNewDatabase(log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	log("⚠️ ATENÇÃO: Iniciando criação de NOVA base (Drop Database)...")

	if err := m.conn.Database.Drop(ctx); err != nil {
		return fmt.Errorf("erro ao dropar base de dados: %w", err)
	}

	log("Base de dados recriada com sucesso!")
	return nil
}


func (m *Manager) CleanDatabaseByDate(beforeDate string, log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	log(fmt.Sprintf("🧹 Limpando movimentações anteriores a %s...", beforeDate))


	date, err := time.Parse("2006-01-02", beforeDate)
	if err != nil {
		return 0, fmt.Errorf("formato de data inválido (use YYYY-MM-DD): %w", err)
	}

	totalDeleted := 0


	collections := map[string]string{
		"Movimentacoes":          "DataMovimentacao",
		"ContasReceber":          "DataEmissao",
		"ContasPagar":            "DataEmissao",
		"DocumentosFiscaisSaida": "DataEmissao",
	}

	for collName, dateField := range collections {
		if m.state.ShouldStop() {
			log("Operação cancelada")
			return totalDeleted, nil
		}

		coll := m.conn.GetCollection(collName)
		filter := bson.M{dateField: bson.M{"$lt": date}}

		result, err := coll.DeleteMany(ctx, filter)
		if err != nil {
			log(fmt.Sprintf("⚠️ Erro em %s: %s", collName, err.Error()))
			continue
		}

		deleted := int(result.DeletedCount)
		totalDeleted += deleted
		if deleted > 0 {
			log(fmt.Sprintf("📦 %s: %d registros removidos", collName, deleted))
		}
	}

	log(fmt.Sprintf("✅ Total: %d registros removidos", totalDeleted))
	return totalDeleted, nil
}

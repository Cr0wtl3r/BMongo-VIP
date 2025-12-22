package operations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"BMongo-VIP/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InactivateZeroProducts deactivates products with zero or negative stock
func (m *Manager) InactivateZeroProducts(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	estoques := m.conn.GetCollection(database.CollectionEstoques)
	produtosServicos := m.conn.GetCollection(database.CollectionProdutosServicos)
	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)

	// Find stocks with zero or negative quantities (NOT fractional like 0.1, 0.5)
	filter := bson.M{
		"$or": []bson.M{
			{"Quantidades.0.Quantidade": bson.M{"$lte": 0}},
			{"Quantidades": bson.A{}},
		},
	}

	cursor, err := estoques.Find(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("erro ao buscar estoques: %w", err)
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		if m.state.ShouldStop() {
			log("Operação cancelada")
			return count, nil
		}

		var estoque bson.M
		if err := cursor.Decode(&estoque); err != nil {
			continue
		}

		estoqueID, ok := estoque["_id"].(primitive.ObjectID)
		if !ok {
			continue
		}

		// Find product linked to this stock
		var produtoEmpresa bson.M
		err := produtosEmpresa.FindOne(ctx, bson.M{"EstoqueReferencia": estoqueID}).Decode(&produtoEmpresa)
		if err != nil {
			continue
		}

		produtoRef, ok := produtoEmpresa["ProdutoServicoReferencia"].(primitive.ObjectID)
		if !ok {
			continue
		}

		// Deactivate the product
		result, err := produtosServicos.UpdateOne(ctx,
			bson.M{"_id": produtoRef},
			bson.M{"$set": bson.M{"Ativo": false}},
		)
		if err != nil {
			log(fmt.Sprintf("Erro ao atualizar produto %s: %s", produtoRef.Hex(), err.Error()))
			continue
		}

		if result.ModifiedCount > 0 {
			count++
			log(fmt.Sprintf("Produto %s inativado", produtoRef.Hex()))
		}
	}

	return count, nil
}

// ChangeTributationByNCM changes tributation for products by NCM prefix
func (m *Manager) ChangeTributationByNCM(ncms []string, tributationID string, log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Validate tributation ID
	tribID, err := primitive.ObjectIDFromHex(tributationID)
	if err != nil {
		return 0, fmt.Errorf("ID de tributação inválido: %w", err)
	}

	// Verify tributation exists
	tributacoes := m.conn.GetCollection(database.CollectionTributacoesEstadual)
	count64, err := tributacoes.CountDocuments(ctx, bson.M{"_id": tribID})
	if err != nil || count64 == 0 {
		return 0, fmt.Errorf("tributação não encontrada")
	}

	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	totalUpdates := 0

	for _, ncm := range ncms {
		if m.state.ShouldStop() {
			log("Operação cancelada")
			return totalUpdates, nil
		}

		ncm = trimSpace(ncm)
		if ncm == "" {
			continue
		}

		// Update products matching NCM prefix
		result, err := produtosEmpresa.UpdateMany(ctx,
			bson.M{"NcmNbs.Codigo": bson.M{"$regex": fmt.Sprintf("^%s.*", ncm), "$options": "i"}},
			bson.M{"$set": bson.M{"TributacaoEstadualReferencia": tribID}},
		)
		if err != nil {
			log(fmt.Sprintf("Erro ao processar NCM %s: %s", ncm, err.Error()))
			continue
		}

		if result.ModifiedCount > 0 {
			totalUpdates += int(result.ModifiedCount)
			log(fmt.Sprintf("Atualizado %d produtos com NCM %s", result.ModifiedCount, ncm))
		} else {
			log(fmt.Sprintf("Nenhum produto encontrado para NCM %s", ncm))
		}
	}

	return totalUpdates, nil
}

// GetTributations returns all active tributations
func (m *Manager) GetTributations() ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tributacoes := m.conn.GetCollection(database.CollectionTributacoesEstadual)

	cursor, err := tributacoes.Find(ctx, bson.M{"Ativo": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var trib bson.M
		if err := cursor.Decode(&trib); err != nil {
			continue
		}

		id := ""
		if oid, ok := trib["_id"].(primitive.ObjectID); ok {
			id = oid.Hex()
		}

		desc := "Sem descrição"
		if d, ok := trib["Descricao"].(string); ok {
			desc = d
		}

		results = append(results, map[string]interface{}{
			"id":        id,
			"Descricao": desc,
		})
	}

	return results, nil
}

// EnableMEI enables MEI stock adjustment
func (m *Manager) EnableMEI(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pessoas := m.conn.GetCollection(database.CollectionPessoas)

	result, err := pessoas.UpdateMany(ctx,
		bson.M{"_t.2": "Emitente"},
		bson.M{"$set": bson.M{"MicroempreendedorIndividual.Habilitado": true}},
	)
	if err != nil {
		return 0, err
	}

	count := int(result.ModifiedCount)
	if count == 0 {
		log("Nenhuma referência encontrada. Verifique a base.")
	} else if count == 1 {
		log(fmt.Sprintf("Foi alterada %d referência", count))
	} else {
		log(fmt.Sprintf("Foram alteradas %d referências", count))
	}

	return count, nil
}

func trimSpace(s string) string {
	// Simple trim implementation
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

// GetFederalTributations returns all federal tributations with only ID and Description
func (m *Manager) GetFederalTributations() ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	collection := m.conn.GetCollection(database.CollectionTributacoesFederal)

	// Projection to return only necessary fields
	opts := options.Find().SetProjection(bson.M{
		"_id":       1,
		"Descricao": 1,
	})

	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar tributações federais: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("erro ao decodificar tributações: %w", err)
	}

	// Convert ObjectID to string for frontend
	for i, doc := range results {
		if oid, ok := doc["_id"].(primitive.ObjectID); ok {
			results[i]["id"] = oid.Hex()
		}
	}

	return results, nil
}

// ChangeFederalTributationByNCM updates the federal tributation for products matching the given NCMs
func (m *Manager) ChangeFederalTributationByNCM(ncms []string, tributationID string, log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if len(ncms) == 0 {
		return fmt.Errorf("nenhum NCM informado")
	}

	tribOID, err := primitive.ObjectIDFromHex(tributationID)
	if err != nil {
		return fmt.Errorf("ID da tributação inválido")
	}

	// Fetch the full tributation object first
	tributacoesFederal := m.conn.GetCollection(database.CollectionTributacoesFederal)
	var tributacaoObj bson.M
	err = tributacoesFederal.FindOne(ctx, bson.M{"_id": tribOID}).Decode(&tributacaoObj)
	if err != nil {
		return fmt.Errorf("erro ao buscar tributação federal: %w", err)
	}

	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)

	// Create Regex pattern: ^(ncm1|ncm2|ncm3)
	pattern := "^(" + strings.Join(ncms, "|") + ")"
	filter := bson.M{
		"NcmNbs.Codigo": bson.M{
			"$regex": primitive.Regex{Pattern: pattern, Options: ""},
		},
	}

	update := bson.M{
		"$set": bson.M{
			"TributacaoFederal":           tributacaoObj,
			"TributacaoFederalReferencia": tribOID,
		},
	}

	log(fmt.Sprintf("Atualizando produtos com NCMs: %s", strings.Join(ncms, ", ")))

	result, err := produtosEmpresa.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("erro ao atualizar produtos: %w", err)
	}

	log(fmt.Sprintf("Sucesso! %d produtos atualizados.", result.ModifiedCount))
	return nil
}

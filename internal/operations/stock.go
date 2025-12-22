package operations

import (
	"BMongo-VIP/internal/database"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// ZeroAllStock sets all product stock quantities to zero
func (m *Manager) ZeroAllStock(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("🔄 Zerando TODO o estoque...")

	estoques := m.conn.GetCollection(database.CollectionEstoques)

	// Update all stock quantities to 0
	result, err := estoques.UpdateMany(ctx,
		bson.M{},
		bson.M{
			"$set": bson.M{
				"Quantidades.0.Quantidade": 0.0,
			},
		},
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao zerar estoque: %w", err)
	}

	count := int(result.ModifiedCount)
	log(fmt.Sprintf("✅ %d estoques zerados", count))
	return count, nil
}

// ZeroNegativeStock sets only negative stock quantities to zero
func (m *Manager) ZeroNegativeStock(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("🔄 Zerando estoques NEGATIVOS...")

	estoques := m.conn.GetCollection(database.CollectionEstoques)

	// Find and update only negative quantities
	filter := bson.M{
		"Quantidades.0.Quantidade": bson.M{"$lt": 0},
	}

	result, err := estoques.UpdateMany(ctx,
		filter,
		bson.M{
			"$set": bson.M{
				"Quantidades.0.Quantidade": 0.0,
			},
		},
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao zerar estoque negativo: %w", err)
	}

	count := int(result.ModifiedCount)
	log(fmt.Sprintf("✅ %d estoques negativos zerados", count))
	return count, nil
}

// ZeroAllPrices sets all product prices (cost and sale) to zero
func (m *Manager) ZeroAllPrices(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("🔄 Zerando TODOS os preços...")

	// Zero prices in ProdutosServicosEmpresa
	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)

	result, err := produtosEmpresa.UpdateMany(ctx,
		bson.M{},
		bson.M{
			"$set": bson.M{
				"PrecosCustos.0.Valor": 0.0,
				"PrecosVendas.0.Valor": 0.0,
			},
		},
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao zerar preços: %w", err)
	}

	count := int(result.ModifiedCount)
	log(fmt.Sprintf("✅ %d preços zerados", count))
	return count, nil
}

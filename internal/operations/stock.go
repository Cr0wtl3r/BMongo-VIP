package operations

import (
	"BMongo-VIP/internal/database"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *Manager) ZeroAllStock(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("🔄 Zerando TODO o estoque...")

	estoques := m.conn.GetCollection(database.CollectionEstoques)

	var stocksBackup []map[string]interface{}
	if m.rollback != nil {
		log("📋 Capturando estoques anteriores para rollback...")
		cursor, err := estoques.Find(ctx, bson.M{"Quantidades.0.Quantidade": bson.M{"$ne": 0}})
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var doc bson.M
				if err := cursor.Decode(&doc); err != nil {
					continue
				}
				id, _ := doc["_id"].(primitive.ObjectID)
				qty := 0.0
				if quantidades, ok := doc["Quantidades"].(primitive.A); ok && len(quantidades) > 0 {
					if q0, ok := quantidades[0].(bson.M); ok {
						qty, _ = q0["Quantidade"].(float64)
					}
				}
				stocksBackup = append(stocksBackup, map[string]interface{}{
					"id":           id.Hex(),
					"prevQuantity": qty,
				})
			}
		}
		log(fmt.Sprintf("📋 Capturados %d estoques para backup", len(stocksBackup)))
	}

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

	if m.rollback != nil && len(stocksBackup) > 0 {
		m.rollback.RecordOperation(
			OpZeroStock,
			fmt.Sprintf("Zerou %d estoques", count),
			map[string]interface{}{"stocks": stocksBackup},
			true,
		)
	}

	log(fmt.Sprintf("✅ %d estoques zerados", count))
	return count, nil
}

func (m *Manager) ZeroNegativeStock(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("🔄 Zerando estoques NEGATIVOS...")

	estoques := m.conn.GetCollection(database.CollectionEstoques)

	filter := bson.M{
		"Quantidades.0.Quantidade": bson.M{"$lt": 0},
	}

	var stocksBackup []map[string]interface{}
	if m.rollback != nil {
		log("📋 Capturando estoques negativos para rollback...")
		cursor, err := estoques.Find(ctx, filter)
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var doc bson.M
				if err := cursor.Decode(&doc); err != nil {
					continue
				}
				id, _ := doc["_id"].(primitive.ObjectID)
				qty := 0.0
				if quantidades, ok := doc["Quantidades"].(primitive.A); ok && len(quantidades) > 0 {
					if q0, ok := quantidades[0].(bson.M); ok {
						qty, _ = q0["Quantidade"].(float64)
					}
				}
				stocksBackup = append(stocksBackup, map[string]interface{}{
					"id":           id.Hex(),
					"prevQuantity": qty,
				})
			}
		}
		log(fmt.Sprintf("📋 Capturados %d estoques negativos para backup", len(stocksBackup)))
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

	if m.rollback != nil && len(stocksBackup) > 0 {
		m.rollback.RecordOperation(
			OpZeroNegativeStock,
			fmt.Sprintf("Zerou %d estoques negativos", count),
			map[string]interface{}{"stocks": stocksBackup},
			true,
		)
	}

	log(fmt.Sprintf("✅ %d estoques negativos zerados", count))
	return count, nil
}

func (m *Manager) ZeroAllPrices(log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("🔄 Zerando TODOS os preços...")

	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)

	var pricesBackup []map[string]interface{}
	if m.rollback != nil {
		log("📋 Capturando preços anteriores para rollback...")
		filter := bson.M{
			"$or": []bson.M{
				{"PrecosCustos.0.Valor": bson.M{"$ne": 0}},
				{"PrecosVendas.0.Valor": bson.M{"$ne": 0}},
			},
		}
		cursor, err := produtosEmpresa.Find(ctx, filter)
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var doc bson.M
				if err := cursor.Decode(&doc); err != nil {
					continue
				}
				id, _ := doc["_id"].(primitive.ObjectID)
				prevCost := 0.0
				prevSale := 0.0

				if custos, ok := doc["PrecosCustos"].(primitive.A); ok && len(custos) > 0 {
					if c0, ok := custos[0].(bson.M); ok {
						prevCost, _ = c0["Valor"].(float64)
					}
				}
				if vendas, ok := doc["PrecosVendas"].(primitive.A); ok && len(vendas) > 0 {
					if v0, ok := vendas[0].(bson.M); ok {
						prevSale, _ = v0["Valor"].(float64)
					}
				}

				pricesBackup = append(pricesBackup, map[string]interface{}{
					"id":       id.Hex(),
					"prevCost": prevCost,
					"prevSale": prevSale,
				})
			}
		}
		log(fmt.Sprintf("📋 Capturados %d preços para backup", len(pricesBackup)))
	}

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

	if m.rollback != nil && len(pricesBackup) > 0 {
		m.rollback.RecordOperation(
			OpZeroAllPrices,
			fmt.Sprintf("Zerou preços de %d produtos", count),
			map[string]interface{}{"products": pricesBackup},
			true,
		)
	}

	log(fmt.Sprintf("✅ %d preços zerados", count))
	return count, nil
}

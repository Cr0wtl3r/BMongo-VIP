package operations

import (
	"BMongo-VIP/internal/database"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)


type ProductFilter struct {
	QuantityOp    string   `json:"quantityOp"`
	QuantityValue float64  `json:"quantityValue"`
	Brand         string   `json:"brand"`
	NCMs          []string `json:"ncms"`
	Weighable     *bool    `json:"weighable"`
	ItemType      string   `json:"itemType"`
	StateTribID   string   `json:"stateTribId"`
	FederalTribID string   `json:"federalTribId"`
	CostPriceOp   string   `json:"costPriceOp"`
	CostPriceVal  float64  `json:"costPriceVal"`
	SalePriceOp   string   `json:"salePriceOp"`
	SalePriceVal  float64  `json:"salePriceVal"`
	ActiveStatus  *bool    `json:"activeStatus"`
}


type FilteredProduct struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Brand     string  `json:"brand"`
	NCM       string  `json:"ncm"`
	Quantity  float64 `json:"quantity"`
	CostPrice float64 `json:"costPrice"`
	SalePrice float64 `json:"salePrice"`
	Active    bool    `json:"active"`
	Weighable bool    `json:"weighable"`
	ItemType  string  `json:"itemType"`
}


type FilterResult struct {
	Products []FilteredProduct `json:"products"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
}


func (m *Manager) FilterProducts(filter ProductFilter, log LogFunc) (FilterResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log("🔍 Aplicando filtros...")


	empresaFilter := bson.M{}


	if len(filter.NCMs) > 0 {
		patterns := make([]bson.M, len(filter.NCMs))
		for i, ncm := range filter.NCMs {
			ncm = trimSpace(ncm)
			if ncm != "" {
				patterns[i] = bson.M{"NcmNbs.Codigo": bson.M{"$regex": "^" + ncm}}
			}
		}
		if len(patterns) > 0 {
			empresaFilter["$or"] = patterns
		}
	}


	if filter.StateTribID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.StateTribID); err == nil {
			empresaFilter["TributacaoEstadualReferencia"] = oid
		}
	}
	if filter.FederalTribID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.FederalTribID); err == nil {
			empresaFilter["TributacaoFederalReferencia"] = oid
		}
	}


	if filter.CostPriceOp != "" && filter.CostPriceVal > 0 {
		empresaFilter["PrecosCustos.0.Valor"] = buildComparisonFilter(filter.CostPriceOp, filter.CostPriceVal)
	}
	if filter.SalePriceOp != "" && filter.SalePriceVal > 0 {
		empresaFilter["PrecosVendas.0.Valor"] = buildComparisonFilter(filter.SalePriceOp, filter.SalePriceVal)
	}


	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)


	totalEmpresaCount, err := produtosEmpresa.CountDocuments(ctx, empresaFilter)
	if err != nil {
		log(fmt.Sprintf("⚠️ Erro ao contar: %v", err))
	}
	log(fmt.Sprintf("📊 Total de produtos no filtro empresa: %d", totalEmpresaCount))


	opts := options.Find().SetLimit(2000)
	cursor, err := produtosEmpresa.Find(ctx, empresaFilter, opts)
	if err != nil {
		return FilterResult{}, fmt.Errorf("erro ao buscar produtos empresa: %w", err)
	}
	defer cursor.Close(ctx)

	var empresaDocs []bson.M
	if err = cursor.All(ctx, &empresaDocs); err != nil {
		return FilterResult{}, fmt.Errorf("erro ao decodificar: %w", err)
	}

	log(fmt.Sprintf("📦 Carregados %d produtos para exibição", len(empresaDocs)))


	produtoRefs := make([]primitive.ObjectID, 0, len(empresaDocs))
	empresaMap := make(map[string]bson.M)
	for _, doc := range empresaDocs {
		if ref, ok := doc["ProdutoServicoReferencia"].(primitive.ObjectID); ok {
			produtoRefs = append(produtoRefs, ref)
			empresaMap[ref.Hex()] = doc
		}
	}


	produtoFilterForCount := bson.M{"_id": bson.M{"$in": produtoRefs}}
	if filter.Brand != "" {
		produtoFilterForCount["Marca.Descricao"] = bson.M{"$regex": filter.Brand, "$options": "i"}
	}
	if filter.Weighable != nil {
		produtoFilterForCount["Pesavel"] = *filter.Weighable
	}
	if filter.ItemType != "" {
		produtoFilterForCount["TipoItem.Descricao"] = bson.M{"$regex": filter.ItemType, "$options": "i"}
	}
	if filter.ActiveStatus != nil {
		produtoFilterForCount["Ativo"] = *filter.ActiveStatus
	}


	produtosServicos := m.conn.GetCollection(database.CollectionProdutosServicos)
	produtoFilter := produtoFilterForCount
	cursor2, err := produtosServicos.Find(ctx, produtoFilter)
	if err != nil {
		return FilterResult{}, fmt.Errorf("erro ao buscar produtos: %w", err)
	}
	defer cursor2.Close(ctx)

	var results []FilteredProduct
	for cursor2.Next(ctx) {
		if m.state.ShouldStop() {
			log("Operação cancelada")
			break
		}

		var produto bson.M
		if err := cursor2.Decode(&produto); err != nil {
			continue
		}

		prodID, _ := produto["_id"].(primitive.ObjectID)
		empresaDoc := empresaMap[prodID.Hex()]


		qty := 0.0
		if empresaDoc != nil {
			if estoqueRef, ok := empresaDoc["EstoqueReferencia"].(primitive.ObjectID); ok {
				qty = m.getStockQuantity(ctx, estoqueRef)
			}
		}


		if filter.QuantityOp != "" {
			if !matchesComparison(qty, filter.QuantityOp, filter.QuantityValue) {
				continue
			}
		}


		brandName := ""
		if marca, ok := produto["Marca"].(bson.M); ok {
			brandName = getString(marca, "Descricao")
		}
		itemTypeName := ""
		if tipoItem, ok := produto["TipoItem"].(bson.M); ok {
			itemTypeName = getString(tipoItem, "Descricao")
		}

		fp := FilteredProduct{
			ID:        prodID.Hex(),
			Name:      getString(produto, "Descricao"),
			Brand:     brandName,
			Active:    getBool(produto, "Ativo"),
			Weighable: getBool(produto, "Pesavel"),
			ItemType:  itemTypeName,
			Quantity:  qty,
		}

		if empresaDoc != nil {
			fp.NCM = getNestedString(empresaDoc, "NcmNbs", "Codigo")
			fp.CostPrice = getNestedFloat(empresaDoc, "PrecosCustos", 0, "Valor")
			fp.SalePrice = getNestedFloat(empresaDoc, "PrecosVendas", 0, "Valor")
		}

		results = append(results, fp)
	}

	log(fmt.Sprintf("✅ %d produtos retornados de %d total no filtro", len(results), totalEmpresaCount))
	return FilterResult{
		Products: results,
		Total:    totalEmpresaCount,
		Limit:    2000,
	}, nil
}


func (m *Manager) BulkActivateProducts(productIDs []string, activate bool, log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	action := "inativando"
	if activate {
		action = "ativando"
	}
	log(fmt.Sprintf("🔄 %s %d produtos...", action, len(productIDs)))

	oids := make([]primitive.ObjectID, 0, len(productIDs))
	for _, id := range productIDs {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			oids = append(oids, oid)
		}
	}

	produtosServicos := m.conn.GetCollection(database.CollectionProdutosServicos)
	result, err := produtosServicos.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": oids}},
		bson.M{"$set": bson.M{"Ativo": activate}},
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao atualizar: %w", err)
	}

	count := int(result.ModifiedCount)
	log(fmt.Sprintf("✅ %d produtos atualizados", count))
	return count, nil
}


func (m *Manager) BulkActivateByFilter(filter ProductFilter, activate bool, log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	action := "inativando"
	if activate {
		action = "ativando"
	}
	log(fmt.Sprintf("🔄 %s TODOS os produtos do filtro...", action))


	empresaFilter := bson.M{}

	if len(filter.NCMs) > 0 {
		patterns := make([]bson.M, 0)
		for _, ncm := range filter.NCMs {
			ncm = trimSpace(ncm)
			if ncm != "" {
				patterns = append(patterns, bson.M{"NcmNbs.Codigo": bson.M{"$regex": "^" + ncm}})
			}
		}
		if len(patterns) > 0 {
			empresaFilter["$or"] = patterns
		}
	}

	if filter.StateTribID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.StateTribID); err == nil {
			empresaFilter["TributacaoEstadualReferencia"] = oid
		}
	}
	if filter.FederalTribID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.FederalTribID); err == nil {
			empresaFilter["TributacaoFederalReferencia"] = oid
		}
	}


	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	cursor, err := produtosEmpresa.Find(ctx, empresaFilter)
	if err != nil {
		return 0, fmt.Errorf("erro ao buscar: %w", err)
	}
	defer cursor.Close(ctx)

	var produtoRefs []primitive.ObjectID
	for cursor.Next(ctx) {
		if m.state.ShouldStop() {
			log("Operação cancelada")
			return 0, nil
		}
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		if ref, ok := doc["ProdutoServicoReferencia"].(primitive.ObjectID); ok {
			produtoRefs = append(produtoRefs, ref)
		}
	}

	log(fmt.Sprintf("📦 %d produtos encontrados no filtro", len(produtoRefs)))

	if len(produtoRefs) == 0 {
		return 0, nil
	}


	produtoFilter := bson.M{"_id": bson.M{"$in": produtoRefs}}

	if filter.Brand != "" {
		produtoFilter["Marca.Descricao"] = bson.M{"$regex": filter.Brand, "$options": "i"}
	}
	if filter.Weighable != nil {
		produtoFilter["Pesavel"] = *filter.Weighable
	}
	if filter.ItemType != "" {
		produtoFilter["TipoItem.Descricao"] = bson.M{"$regex": filter.ItemType, "$options": "i"}
	}
	if filter.ActiveStatus != nil {
		produtoFilter["Ativo"] = *filter.ActiveStatus
	}


	produtosServicos := m.conn.GetCollection(database.CollectionProdutosServicos)
	result, err := produtosServicos.UpdateMany(ctx,
		produtoFilter,
		bson.M{"$set": bson.M{"Ativo": activate}},
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao atualizar: %w", err)
	}

	count := int(result.ModifiedCount)
	log(fmt.Sprintf("✅ %d produtos atualizados no total!", count))
	return count, nil
}


func buildComparisonFilter(op string, value float64) bson.M {
	switch op {
	case "gt":
		return bson.M{"$gt": value}
	case "lt":
		return bson.M{"$lt": value}
	case "gte":
		return bson.M{"$gte": value}
	case "lte":
		return bson.M{"$lte": value}
	case "eq":
		return bson.M{"$eq": value}
	default:
		return bson.M{"$gte": 0}
	}
}

func matchesComparison(value float64, op string, target float64) bool {
	switch op {
	case "gt":
		return value > target
	case "lt":
		return value < target
	case "gte":
		return value >= target
	case "lte":
		return value <= target
	case "eq":
		return value == target
	default:
		return true
	}
}

func (m *Manager) getStockQuantity(ctx context.Context, estoqueID primitive.ObjectID) float64 {
	estoques := m.conn.GetCollection(database.CollectionEstoques)
	var estoque bson.M
	err := estoques.FindOne(ctx, bson.M{"_id": estoqueID}).Decode(&estoque)
	if err != nil {
		return 0
	}

	if quantidades, ok := estoque["Quantidades"].(primitive.A); ok && len(quantidades) > 0 {
		if q0, ok := quantidades[0].(bson.M); ok {
			if qty, ok := q0["Quantidade"].(float64); ok {
				return qty
			}
		}
	}
	return 0
}

func getString(doc bson.M, field string) string {
	if v, ok := doc[field].(string); ok {
		return v
	}
	return ""
}

func getBool(doc bson.M, field string) bool {
	if v, ok := doc[field].(bool); ok {
		return v
	}
	return false
}

func getNestedString(doc bson.M, field1, field2 string) string {
	if nested, ok := doc[field1].(bson.M); ok {
		if v, ok := nested[field2].(string); ok {
			return v
		}
	}
	return ""
}

func getNestedFloat(doc bson.M, field string, index int, subfield string) float64 {
	if arr, ok := doc[field].(primitive.A); ok && len(arr) > index {
		if item, ok := arr[index].(bson.M); ok {
			if v, ok := item[subfield].(float64); ok {
				return v
			}
		}
	}
	return 0
}

package operations

import (
	"BMongo-VIP/internal/database"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductFilter struct {
	// Texto (contém)
	Name         string `json:"name"`
	InternalCode string `json:"internalCode"`
	Barcode      string `json:"barcode"`

	// NCM
	NCMs []string `json:"ncms"`

	// Marca
	Brand   string `json:"brand"`
	BrandID string `json:"brandId"`

	// Tributações (por ID)
	StateTribID     string `json:"stateTribId"`
	FederalTribID   string `json:"federalTribId"`
	MunicipalTribID string `json:"municipalTribId"`
	IbsCbsTribID    string `json:"ibsCbsTribId"`

	// Tipo de Item Fiscal (Mercadoria para revenda, Matéria Prima, etc)
	ItemType   string `json:"itemType"`
	ItemTypeID string `json:"itemTypeId"`

	// Gênero do Item / Tipo do Produto (Serviço, Produto, GLP, Vasilhame, etc)
	ProductType   string `json:"productType"`
	ProductTypeID string `json:"productTypeId"`

	// Código de Atividade (Serviços) - Multiple IDs
	ActivityCodes string `json:"activityCodes"`

	// Código Tributação Município (filter services that have it filled)
	HasCodigoTribMunicipio *bool `json:"hasCodigoTribMunicipio"`

	// Status
	ActiveStatus *bool `json:"activeStatus"`
	Weighable    *bool `json:"weighable"`

	// Quantidade
	QuantityOp    string  `json:"quantityOp"`
	QuantityValue float64 `json:"quantityValue"`

	// Preços
	CostPriceOp  string  `json:"costPriceOp"`
	CostPriceVal float64 `json:"costPriceVal"`
	SalePriceOp  string  `json:"salePriceOp"`
	SalePriceVal float64 `json:"salePriceVal"`
}

type FilteredProduct struct {
	ID              string  `json:"id"`
	EmpresaID       string  `json:"empresaId"`
	Name            string  `json:"name"`
	InternalCode    string  `json:"internalCode"`
	Barcode         string  `json:"barcode"`
	Brand           string  `json:"brand"`
	BrandID         string  `json:"brandId"`
	NCM             string  `json:"ncm"`
	StateTribID     string  `json:"stateTribId"`
	StateTribName   string  `json:"stateTribName"`
	FederalTribID   string  `json:"federalTribId"`
	FederalTribName string  `json:"federalTribName"`
	MunicipalTribID string  `json:"municipalTribId"`
	IbsCbsTribID    string  `json:"ibsCbsTribId"`
	ItemType        string  `json:"itemType"`
	ItemTypeID      string  `json:"itemTypeId"`
	ProductType     string  `json:"productType"`
	ProductTypeID   string  `json:"productTypeId"`
	Quantity        float64 `json:"quantity"`
	CostPrice       float64 `json:"costPrice"`
	SalePrice       float64 `json:"salePrice"`
	Active          bool    `json:"active"`
	Weighable       bool    `json:"weighable"`
	PrecoRefID      string  `json:"-"`
	EstoqueRefID    string  `json:"-"`
}

type FilterResult struct {
	Products []FilteredProduct `json:"products"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
}

func (m *Manager) FilterProducts(filter ProductFilter, log LogFunc) (FilterResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log("🔍 Aplicando filtros (Lógica Híbrida)...")

	// 1. Build Base Filters
	empresaFilter := m.buildEmpresaFilter(filter)
	produtoFilter := m.buildProdutoFilter(filter)

	// DEBUG LOGS
	log(fmt.Sprintf("🔍 Filtro Recebido - Name: '%s', Code: '%s', Barcode: '%s'", filter.Name, filter.InternalCode, filter.Barcode))
	codes := splitAndTrim(filter.InternalCode)
	barcodes := splitAndTrim(filter.Barcode)
	if len(codes) > 0 || len(barcodes) > 0 {
		log(fmt.Sprintf("🔢 Codes: %v, Barcodes: %v", codes, barcodes))
	}

	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	produtosServicos := m.conn.GetCollection(database.CollectionProdutosServicos)

	// 2. Decide Strategy
	// Strategy A: Product-First (Best for specific searches like Code, Name, Brand, Type)
	// Strategy B: Company-First (Best for general browsing like "Active", "Price", or Empty)

	hasProductFilters := filter.Name != "" || filter.InternalCode != "" || filter.Barcode != "" ||
		filter.Brand != "" || filter.BrandID != "" ||
		filter.ItemType != "" || filter.ItemTypeID != "" ||
		filter.ProductType != "" || filter.ProductTypeID != ""

	var results []FilteredProduct
	empresaMap := make(map[string]bson.M)

	if hasProductFilters {
		log("🚀 Estratégia: BUSCA POR PRODUTO PRIMEIRO")

		// Step 1: Find matching products (limit 2000 to prevent overflow on broad text searches)
		optsProd := options.Find().SetLimit(2000)
		cursorProd, err := produtosServicos.Find(ctx, produtoFilter, optsProd)
		if err != nil {
			return FilterResult{}, fmt.Errorf("erro ao buscar produtos (strategy A): %w", err)
		}
		defer cursorProd.Close(ctx)

		var matchedProducts []bson.M
		if err = cursorProd.All(ctx, &matchedProducts); err != nil {
			return FilterResult{}, fmt.Errorf("erro ao decodificar produtos: %w", err)
		}

		if len(matchedProducts) == 0 {
			log("❌ Nenhum produto encontrado na busca primária.")
			return FilterResult{Products: []FilteredProduct{}, Total: 0, Limit: 2000}, nil
		}

		log(fmt.Sprintf("📦 Encontrados %d produtos base. Filtrando disponibilidade na empresa...", len(matchedProducts)))

		// Step 2: Extract IDs and Filter by Empresa availability
		productIDs := make([]primitive.ObjectID, 0, len(matchedProducts))
		productMap := make(map[string]bson.M)
		for _, p := range matchedProducts {
			if id, ok := p["_id"].(primitive.ObjectID); ok {
				productIDs = append(productIDs, id)
				productMap[id.Hex()] = p
			}
		}

		// Add restriction to empresaFilter
		if len(productIDs) > 0 {
			empresaFilter["ProdutoServicoReferencia"] = bson.M{"$in": productIDs}
		}

		// Fetch from Empresa
		cursorEmp, err := produtosEmpresa.Find(ctx, empresaFilter) // Valid filters + ID restriction
		if err != nil {
			return FilterResult{}, fmt.Errorf("erro ao cruzar com empresa: %w", err)
		}
		defer cursorEmp.Close(ctx)

		for cursorEmp.Next(ctx) {
			var empDoc bson.M
			if err := cursorEmp.Decode(&empDoc); err != nil {
				continue
			}

			// Join Logic
			if ref, ok := empDoc["ProdutoServicoReferencia"].(primitive.ObjectID); ok {
				if prodDoc, exists := productMap[ref.Hex()]; exists {
					results = append(results, m.mapToFilteredProduct(prodDoc, empDoc))
				}
			}
		}

	} else {
		log("🏢 Estratégia: BUSCA POR EMPRESA PRIMEIRO (Filtros Genéricos)")

		// Step 1: Find company docs (Limit 2000)
		optsEmp := options.Find().SetLimit(2000)
		cursorEmp, err := produtosEmpresa.Find(ctx, empresaFilter, optsEmp)
		if err != nil {
			return FilterResult{}, fmt.Errorf("erro ao buscar produtos empresa: %w", err)
		}
		defer cursorEmp.Close(ctx)

		var empresaDocs []bson.M
		if err = cursorEmp.All(ctx, &empresaDocs); err != nil {
			return FilterResult{}, fmt.Errorf("erro ao decodificar empresa: %w", err)
		}

		log(fmt.Sprintf("📦 Carregados %d produtos empresa. Aplicando filtro de produto...", len(empresaDocs)))

		// Create Map and ID list
		produtoRefs := make([]primitive.ObjectID, 0, len(empresaDocs))
		for _, doc := range empresaDocs {
			if ref, ok := doc["ProdutoServicoReferencia"].(primitive.ObjectID); ok {
				produtoRefs = append(produtoRefs, ref)
				empresaMap[ref.Hex()] = doc
			}
		}

		if len(produtoRefs) > 0 {
			produtoFilter["_id"] = bson.M{"$in": produtoRefs}
		} else {
			return FilterResult{Products: []FilteredProduct{}, Total: 0, Limit: 2000}, nil
		}

		// Step 2: Find Products matching
		cursorProd, err := produtosServicos.Find(ctx, produtoFilter)
		if err != nil {
			return FilterResult{}, fmt.Errorf("erro ao buscar produtos: %w", err)
		}
		defer cursorProd.Close(ctx)

		for cursorProd.Next(ctx) {
			var prodDoc bson.M
			if err := cursorProd.Decode(&prodDoc); err != nil {
				continue
			}

			prodID, _ := prodDoc["_id"].(primitive.ObjectID)
			if empDoc, exists := empresaMap[prodID.Hex()]; exists {
				results = append(results, m.mapToFilteredProduct(prodDoc, empDoc))
			}
		}
	}

	// Enrich with Price and Stock if needed
	m.enrichProducts(ctx, results)

	log(fmt.Sprintf("✅ %d produtos retornados de %d total no filtro", len(results), len(results)))
	return FilterResult{
		Products: results,
		Total:    int64(len(results)), // Returning the actual count of results found
		Limit:    2000,
	}, nil
}

func (m *Manager) mapToFilteredProduct(produto, empresaDoc bson.M) FilteredProduct {
	// Extract basic fields
	prodID, _ := produto["_id"].(primitive.ObjectID)

	internalCode := getString(produto, "CodigoInterno")
	barcode := getString(produto, "CodigoBarras")

	brandID := ""
	brandName := ""
	if marcaRef, ok := produto["MarcaReferencia"].(primitive.ObjectID); ok {
		brandID = marcaRef.Hex()
	}
	if marca, ok := produto["Marca"].(bson.M); ok {
		brandName = getString(marca, "Descricao")
		// Fallback: If no MarcaReferencia, try to get ID from nested Marca object
		if brandID == "" {
			if id, ok := marca["_id"].(primitive.ObjectID); ok {
				brandID = id.Hex()
			}
		}
	}

	itemTypeID := ""
	itemTypeName := ""
	if tipoRef, ok := produto["TipoItemReferencia"].(primitive.ObjectID); ok {
		itemTypeID = tipoRef.Hex()
	}
	if tipoItem, ok := produto["TipoItem"].(bson.M); ok {
		itemTypeName = getString(tipoItem, "Descricao")
	}

	productTypeID := ""
	productTypeName := ""
	if generoRef, ok := produto["GeneroItemReferencia"].(primitive.ObjectID); ok {
		productTypeID = generoRef.Hex()
	}
	if generoItem, ok := produto["GeneroItem"].(bson.M); ok {
		productTypeName = getString(generoItem, "Descricao")
	}

	// Get stock quantity (need context but helper prevents it easily without passing ctx everywhere)
	// For now, let's assume we can get it from 'EstoqueAtual' if present in empresaDoc,
	// or we need to pass context.
	// Since I cannot change the signature easily in this edit without ensuring I pass ctx in call sites.
	// I DID pass ctx in the calls above? Let's check the previous edit.
	// No, I called `m.mapToFilteredProduct(prodDoc, empDoc)`. I missed ctx.

	// FIX: I will use a simplified quantity extraction if possible, or I need to fix the call sites too.
	// The original code used `m.getStockQuantity(ctx, estoqueRef)`.
	// I will just use `EstoqueAtual` from empresaDoc if available, as 'getStockQuantity' just checks the collection if not embedded.
	// Actually, usually `EstoqueAtual` is in the doc.

	// Get stock quantity (requires context and manager access)
	// We need to fetch it from the database if not present in the doc
	qty := 0.0
	if empresaDoc != nil {
		// Try to get from embedded field first (optimization)
		if q, ok := empresaDoc["EstoqueAtual"]; ok {
			qty = toFloat64(q)
		} else if estoqueRef, ok := empresaDoc["EstoqueReferencia"].(primitive.ObjectID); ok {
			// Fallback to fetching from Estoques collection
			// We need context here. Creating a temporary context for this fetch.
			// This is necessary because helper signature doesn't take context,
			// and refactoring it everywhere takes more steps.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			qty = m.getStockQuantity(ctx, estoqueRef)
		}
	}

	fp := FilteredProduct{
		ID:            prodID.Hex(),
		Name:          getString(produto, "Descricao"),
		InternalCode:  internalCode,
		Barcode:       barcode,
		Brand:         brandName,
		BrandID:       brandID,
		Active:        getBool(produto, "Ativo"),
		Weighable:     getBool(produto, "Pesavel"),
		ItemType:      itemTypeName,
		ItemTypeID:    itemTypeID,
		ProductType:   productTypeName,
		ProductTypeID: productTypeID,
		Quantity:      qty,
	}

	if empresaDoc != nil {
		empID, _ := empresaDoc["_id"].(primitive.ObjectID)
		fp.EmpresaID = empID.Hex()
		fp.NCM = getNestedString(empresaDoc, "NcmNbs", "Codigo")
		fp.CostPrice = getNestedFloat(empresaDoc, "PrecosCustos", 0, "Valor")
		fp.SalePrice = getNestedFloat(empresaDoc, "PrecosVendas", 0, "Valor")

		// DEBUG: Log price data just to see what's inside
		// fmt.Printf("DEBUG: %s -> Cost: %v, Sale: %v, RawCost: %v, RawSale: %v\n",
		// 	fp.InternalCode, fp.CostPrice, fp.SalePrice, empresaDoc["PrecosCustos"], empresaDoc["PrecosVendas"])

		if tribRef, ok := empresaDoc["TributacaoEstadualReferencia"].(primitive.ObjectID); ok {
			fp.StateTribID = tribRef.Hex()
		}
		if tribRef, ok := empresaDoc["TributacaoFederalReferencia"].(primitive.ObjectID); ok {
			fp.FederalTribID = tribRef.Hex()
		}
		if tribRef, ok := empresaDoc["TributacaoIbsCbsReferencia"].(primitive.ObjectID); ok {
			fp.IbsCbsTribID = tribRef.Hex()
		}
		if tribRef, ok := empresaDoc["TributacaoMunicipalReferencia"].(primitive.ObjectID); ok {
			fp.MunicipalTribID = tribRef.Hex()
		}

		// Reference IDs for enrichment
		if ref, ok := empresaDoc["PrecoReferencia"].(primitive.ObjectID); ok {
			fp.PrecoRefID = ref.Hex()
		}
		if ref, ok := empresaDoc["EstoqueReferencia"].(primitive.ObjectID); ok {
			fp.EstoqueRefID = ref.Hex()
		}
	}

	return fp
}

// CountFilteredProducts counts all products matching the filter (no limit)
func (m *Manager) CountFilteredProducts(filter ProductFilter) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	empresaFilter := m.buildEmpresaFilter(filter)
	produtoFilter := m.buildProdutoFilter(filter)

	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	produtosServicos := m.conn.GetCollection(database.CollectionProdutosServicos)

	// Get all matching empresa refs
	cursor, err := produtosEmpresa.Find(ctx, empresaFilter, options.Find().SetProjection(bson.M{"ProdutoServicoReferencia": 1}))
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var produtoRefs []primitive.ObjectID
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		if ref, ok := doc["ProdutoServicoReferencia"].(primitive.ObjectID); ok {
			produtoRefs = append(produtoRefs, ref)
		}
	}

	if len(produtoRefs) == 0 {
		return 0, nil
	}

	produtoFilter["_id"] = bson.M{"$in": produtoRefs}
	return produtosServicos.CountDocuments(ctx, produtoFilter)
}

// GetAllFilteredProductIDs returns all product IDs matching the filter (for bulk operations)
func (m *Manager) GetAllFilteredProductIDs(filter ProductFilter, log LogFunc) ([]string, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log("📋 Obtendo todos os IDs filtrados...")

	empresaFilter := m.buildEmpresaFilter(filter)
	produtoFilter := m.buildProdutoFilter(filter)

	produtosEmpresa := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	produtosServicos := m.conn.GetCollection(database.CollectionProdutosServicos)

	// Get all matching empresa docs
	cursor, err := produtosEmpresa.Find(ctx, empresaFilter, options.Find().SetProjection(bson.M{
		"_id":                      1,
		"ProdutoServicoReferencia": 1,
	}))
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	var produtoRefs []primitive.ObjectID
	empresaIDMap := make(map[string]string) // produtoID -> empresaID

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		empID, _ := doc["_id"].(primitive.ObjectID)
		if ref, ok := doc["ProdutoServicoReferencia"].(primitive.ObjectID); ok {
			produtoRefs = append(produtoRefs, ref)
			empresaIDMap[ref.Hex()] = empID.Hex()
		}
	}

	if len(produtoRefs) == 0 {
		return []string{}, []string{}, nil
	}

	produtoFilter["_id"] = bson.M{"$in": produtoRefs}

	// Get filtered products
	cursor2, err := produtosServicos.Find(ctx, produtoFilter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, nil, err
	}
	defer cursor2.Close(ctx)

	var productIDs []string
	var empresaIDs []string

	for cursor2.Next(ctx) {
		if m.state.ShouldStop() {
			log("Operação cancelada")
			return productIDs, empresaIDs, nil
		}

		var doc bson.M
		if err := cursor2.Decode(&doc); err != nil {
			continue
		}
		if id, ok := doc["_id"].(primitive.ObjectID); ok {
			productIDs = append(productIDs, id.Hex())
			if empID, found := empresaIDMap[id.Hex()]; found {
				empresaIDs = append(empresaIDs, empID)
			}
		}
	}

	log(fmt.Sprintf("📦 %d produtos encontrados", len(productIDs)))
	return productIDs, empresaIDs, nil
}

// buildEmpresaFilter builds MongoDB filter for ProdutosServicosEmpresa collection
func (m *Manager) buildEmpresaFilter(filter ProductFilter) bson.M {
	empresaFilter := bson.M{}

	// NCM filter
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

	// Tributation filters
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
	if filter.MunicipalTribID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.MunicipalTribID); err == nil {
			empresaFilter["TributacaoMunicipalReferencia"] = oid
		}
	}
	if filter.IbsCbsTribID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.IbsCbsTribID); err == nil {
			empresaFilter["TributacaoIbsCbsReferencia"] = oid
		}
	}

	// Price filters
	if filter.CostPriceOp != "" && filter.CostPriceVal > 0 {
		empresaFilter["PrecosCustos.0.Valor"] = buildComparisonFilter(filter.CostPriceOp, filter.CostPriceVal)
	}
	if filter.SalePriceOp != "" && filter.SalePriceVal > 0 {
		empresaFilter["PrecosVendas.0.Valor"] = buildComparisonFilter(filter.SalePriceOp, filter.SalePriceVal)
	}

	// CodigoTributacaoMunicipio filter (services only)
	if filter.HasCodigoTribMunicipio != nil && *filter.HasCodigoTribMunicipio {
		empresaFilter["CodigoTributacaoMunicipio"] = bson.M{
			"$exists": true,
			"$ne":     "",
		}
	}

	return empresaFilter
}

// buildProdutoFilter builds MongoDB filter for ProdutosServicos collection
func (m *Manager) buildProdutoFilter(filter ProductFilter) bson.M {
	produtoFilter := bson.M{}

	// Name filter (contains)
	if filter.Name != "" {
		produtoFilter["Descricao"] = bson.M{"$regex": filter.Name, "$options": "i"}
	}

	// Internal code filter
	internalCodes := splitAndTrim(filter.InternalCode)
	if len(internalCodes) > 0 {
		if len(internalCodes) == 1 {
			produtoFilter["CodigoInterno"] = bson.M{"$regex": regexp.QuoteMeta(internalCodes[0]), "$options": "i"}
		} else {
			regexes := make([]primitive.Regex, len(internalCodes))
			for i, c := range internalCodes {
				regexes[i] = primitive.Regex{Pattern: regexp.QuoteMeta(c), Options: "i"}
			}
			produtoFilter["CodigoInterno"] = bson.M{"$in": regexes}
		}
	}

	// Barcode filter
	barcodes := splitAndTrim(filter.Barcode)
	if len(barcodes) > 0 {
		if len(barcodes) == 1 {
			produtoFilter["CodigoBarras"] = bson.M{"$regex": regexp.QuoteMeta(barcodes[0]), "$options": "i"}
		} else {
			regexes := make([]primitive.Regex, len(barcodes))
			for i, c := range barcodes {
				regexes[i] = primitive.Regex{Pattern: regexp.QuoteMeta(c), Options: "i"}
			}
			produtoFilter["CodigoBarras"] = bson.M{"$in": regexes}
		}
	}

	// Brand filter
	if filter.Brand != "" {
		produtoFilter["Marca.Descricao"] = bson.M{"$regex": filter.Brand, "$options": "i"}
	}
	if filter.BrandID != "" {
		if oid, err := primitive.ObjectIDFromHex(filter.BrandID); err == nil {
			// Search in both locations: MarcaReferencia OR Marca._id
			produtoFilter["$or"] = []bson.M{
				{"MarcaReferencia": oid},
				{"Marca._id": oid},
			}
		}
	}

	// Activity Code filter (for services)
	activityCodes := splitAndTrim(filter.ActivityCodes)
	if len(activityCodes) > 0 {
		var codes []interface{}
		for _, c := range activityCodes {
			// Try as int
			if val, err := strconv.Atoi(c); err == nil {
				codes = append(codes, val)
			}
			// Keep as string too (some codes like "01", "1.02" are strings)
			codes = append(codes, c)
		}
		if len(codes) > 0 {
			produtoFilter["CodigoAtividade.Codigo"] = bson.M{"$in": codes}
		}
	}

	// Item type filter (Fiscal - Mercadoria para revenda, etc)
	if filter.ItemType != "" {
		produtoFilter["TipoItem.Descricao"] = bson.M{"$regex": filter.ItemType, "$options": "i"}
	}
	if filter.ItemTypeID != "" {
		// ItemTypeID is a numeric string like "1", "2", etc. - filter by TipoItem.Codigo
		if codigo, err := strconv.Atoi(filter.ItemTypeID); err == nil {
			produtoFilter["TipoItem.Codigo"] = codigo
		}
	}

	// Product type filter (GeneroItem - Serviço, Produto, GLP, Vasilhame, etc)
	if filter.ProductType != "" {
		if strings.EqualFold(filter.ProductType, "Produto") {
			produtoFilter["_t"] = "Produto"
		} else if strings.EqualFold(filter.ProductType, "Serviço") || strings.EqualFold(filter.ProductType, "Servico") {
			produtoFilter["_t"] = "Servico"
		} else {
			produtoFilter["GeneroItem.Descricao"] = bson.M{"$regex": filter.ProductType, "$options": "i"}
		}
	}
	if filter.ProductTypeID != "" {
		// ProductTypeID is a numeric string like "1", "2", etc.
		if codigo, err := strconv.Atoi(filter.ProductTypeID); err == nil {
			// Special handling for Produto (1) and Serviço (2) using discriminators
			if codigo == 1 {
				produtoFilter["_t"] = "Produto" // or check if it contains "Produto"
			} else if codigo == 2 {
				// Often "Servico" or "ServicoEmpresa" or just check if it contains "Servico"
				// Based on user snippet: "Produto" is in _t.
				// Let's assume standard discriminator query works by checking if array contains value
				produtoFilter["_t"] = "Servico"
			} else {
				// Fallback to Code for others
				produtoFilter["GeneroItem.Codigo"] = codigo
			}
		}
	}

	// Status filters
	if filter.ActiveStatus != nil {
		produtoFilter["Ativo"] = *filter.ActiveStatus
	}
	if filter.Weighable != nil {
		produtoFilter["Pesavel"] = *filter.Weighable
	}

	return produtoFilter
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

	var productsBackup []map[string]interface{}
	if m.rollback != nil {
		cursor, err := produtosServicos.Find(ctx, bson.M{"_id": bson.M{"$in": oids}})
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var doc bson.M
				if err := cursor.Decode(&doc); err != nil {
					continue
				}
				id, _ := doc["_id"].(primitive.ObjectID)
				wasActive, _ := doc["Ativo"].(bool)
				productsBackup = append(productsBackup, map[string]interface{}{
					"id":        id.Hex(),
					"wasActive": wasActive,
				})
			}
		}
	}

	result, err := produtosServicos.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": oids}},
		bson.M{"$set": bson.M{"Ativo": activate}},
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao atualizar: %w", err)
	}

	count := int(result.ModifiedCount)

	if m.rollback != nil && len(productsBackup) > 0 {
		actionLabel := "Inativou"
		if activate {
			actionLabel = "Ativou"
		}
		m.rollback.RecordOperation(
			OpBulkActivate,
			fmt.Sprintf("%s %d produtos", actionLabel, count),
			map[string]interface{}{"products": productsBackup},
			true,
		)
	}

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

	var productsBackup []map[string]interface{}
	if m.rollback != nil {
		backupCursor, err := produtosServicos.Find(ctx, produtoFilter)
		if err == nil {
			defer backupCursor.Close(ctx)
			for backupCursor.Next(ctx) {
				var doc bson.M
				if err := backupCursor.Decode(&doc); err != nil {
					continue
				}
				id, _ := doc["_id"].(primitive.ObjectID)
				wasActive, _ := doc["Ativo"].(bool)
				productsBackup = append(productsBackup, map[string]interface{}{
					"id":        id.Hex(),
					"wasActive": wasActive,
				})
			}
		}
	}

	result, err := produtosServicos.UpdateMany(ctx,
		produtoFilter,
		bson.M{"$set": bson.M{"Ativo": activate}},
	)

	if err != nil {
		return 0, fmt.Errorf("erro ao atualizar: %w", err)
	}

	count := int(result.ModifiedCount)

	if m.rollback != nil && len(productsBackup) > 0 {
		actionLabel := "Inativou"
		if activate {
			actionLabel = "Ativou"
		}
		m.rollback.RecordOperation(
			OpBulkActivate,
			fmt.Sprintf("%s %d produtos (filtro)", actionLabel, count),
			map[string]interface{}{"products": productsBackup},
			true,
		)
	}

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

func (m *Manager) enrichProducts(ctx context.Context, products []FilteredProduct) {
	if len(products) == 0 {
		return
	}

	// 1. Collect unique IDs for Prices and Stocks
	priceIDs := make([]primitive.ObjectID, 0)
	stockIDs := make([]primitive.ObjectID, 0)
	priceMapRef := make(map[string]bool)
	stockMapRef := make(map[string]bool)

	for _, p := range products {
		if (p.CostPrice == 0 || p.SalePrice == 0) && p.PrecoRefID != "" {
			if !priceMapRef[p.PrecoRefID] {
				if oid, err := primitive.ObjectIDFromHex(p.PrecoRefID); err == nil {
					priceIDs = append(priceIDs, oid)
					priceMapRef[p.PrecoRefID] = true
				}
			}
		}
		if p.Quantity == 0 && p.EstoqueRefID != "" {
			if !stockMapRef[p.EstoqueRefID] {
				if oid, err := primitive.ObjectIDFromHex(p.EstoqueRefID); err == nil {
					stockIDs = append(stockIDs, oid)
					stockMapRef[p.EstoqueRefID] = true
				}
			}
		}
	}

	// 2. Fetch Prices
	if len(priceIDs) > 0 {
		coll := m.conn.GetCollection(database.CollectionPrecos)
		cursor, err := coll.Find(ctx, bson.M{"_id": bson.M{"$in": priceIDs}})
		if err == nil {
			defer cursor.Close(ctx)
			fetchedPrices := make(map[string]bson.M)
			for cursor.Next(ctx) {
				var doc bson.M
				if err := cursor.Decode(&doc); err == nil {
					id, _ := doc["_id"].(primitive.ObjectID)
					fetchedPrices[id.Hex()] = doc
					// DEBUG
					// if len(fetchedPrices) == 1 {
					// 	fmt.Printf("DEBUG PRICE DOC: %+v\n", doc)
					// }
				}
			}
			// DEBUG
			fmt.Printf("Fetched %d price documents\n", len(fetchedPrices))
			if len(fetchedPrices) > 0 {
				for _, d := range fetchedPrices {
					fmt.Printf("Sample Price Doc: %+v\n", d)
					break
				}
			}

			// Apply to products
			for i := range products {
				p := &products[i]
				if p.PrecoRefID != "" {
					if doc, found := fetchedPrices[p.PrecoRefID]; found {
						// Try different structures
						if p.CostPrice == 0 {
							p.CostPrice = getNestedFloat(doc, "Custos", 0, "Valor")
						}
						if p.SalePrice == 0 {
							p.SalePrice = getNestedFloat(doc, "Vendas", 0, "Valor")
						}
					}
				}
			}
		}
	}

	// 3. Fetch Stocks
	if len(stockIDs) > 0 {
		coll := m.conn.GetCollection(database.CollectionEstoquesQuantidade)
		cursor, err := coll.Find(ctx, bson.M{"_id": bson.M{"$in": stockIDs}})
		if err == nil {
			defer cursor.Close(ctx)
			fetchedStocks := make(map[string]bson.M)
			for cursor.Next(ctx) {
				var doc bson.M
				if err := cursor.Decode(&doc); err == nil {
					id, _ := doc["_id"].(primitive.ObjectID)
					fetchedStocks[id.Hex()] = doc
				}
			}
			// DEBUG
			fmt.Printf("Fetched %d stock documents\n", len(fetchedStocks))
			if len(fetchedStocks) > 0 {
				for _, d := range fetchedStocks {
					fmt.Printf("Sample Stock Doc: %+v\n", d)
					break
				}
			}

			// Apply to products
			for i := range products {
				p := &products[i]
				if p.EstoqueRefID != "" {
					if doc, found := fetchedStocks[p.EstoqueRefID]; found {
						if p.Quantity == 0 {
							p.Quantity = getNestedFloat(doc, "Quantidades", 0, "Quantidade")
						}
					}
				}
			}
		}
	}
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
			return toFloat64(item[subfield])
		}
	}
	// Also try generic slice interface
	if arr, ok := doc[field].([]interface{}); ok && len(arr) > index {
		if item, ok := arr[index].(map[string]interface{}); ok {
			return toFloat64(item[subfield])
		}
		if item, ok := arr[index].(bson.M); ok {
			return toFloat64(item[subfield])
		}
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case primitive.Decimal128:
		f := val.String()
		if fval, err := strconv.ParseFloat(f, 64); err == nil {
			return fval
		}
	}
	return 0
}

// splitAndTrim splits string by comma or semicolon and trims spaces
func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	// Replace all semicolons with commas to unify splitting
	s = strings.ReplaceAll(s, ";", ",")
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

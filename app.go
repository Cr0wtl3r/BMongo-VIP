package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"BMongo-VIP/internal/database"
	"BMongo-VIP/internal/operations"
	"BMongo-VIP/internal/windows"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	db            *database.Connection
	operations    *operations.Manager
	rollback      *operations.RollbackManager
	numberManager *operations.NumberManager
	logs          []string
	senhaHasheada string
}

func init() {
	if compiledPasswordHash == "" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Aviso: .env não carregado ou não encontrado. Tentando usar variáveis de ambiente ou senha compilada.")
		}
	}
}

func NewApp() *App {
	var hashSenha string
	if compiledPasswordHash != "" {
		hashSenha = compiledPasswordHash
	} else {
		hashSenha = os.Getenv("PASSWORD")
		if hashSenha == "" {
			log.Println("ERRO: PASSWORD (hash) não definido no .env ou como variável de ambiente. O login falhará.")
		}
	}

	return &App{
		logs:          make([]string, 0),
		senhaHasheada: hashSenha,
		numberManager: operations.NewNumberManager(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	conn, err := database.Connect()
	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return
	}

	a.db = conn
	a.operations = operations.NewManager(conn)
	a.rollback = operations.NewRollbackManager(conn)
	a.operations.SetRollback(a.rollback)

	validator := database.NewValidator(conn)
	ok, msg := validator.ValidateConnection()
	if ok {
		a.addLog(fmt.Sprintf("%s", msg))

		empty, _ := validator.IsDatabaseEmpty()
		if empty {
			a.addLog("O banco de dados está vazio. Por favor, restaure uma base.")
		}
	} else {
		a.addLog(fmt.Sprintf("%s", msg))
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Disconnect()
	}
}

func (a *App) Login(senha string) bool {
	hasher := sha256.New()
	hasher.Write([]byte(senha))
	hashSenhaDigitada := hex.EncodeToString(hasher.Sum(nil))
	return hashSenhaDigitada == a.senhaHasheada
}

func (a *App) addLog(message string) {
	a.logs = append(a.logs, message)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "log", message)
	}
}

func (a *App) GetLogs() []string {
	return a.logs
}

func (a *App) ClearLogs() {
	a.logs = make([]string, 0)
}

func (a *App) CheckConnection() bool {
	if a.db == nil {
		return false
	}
	return a.db.IsConnected()
}

func (a *App) RetryConnection() error {
	a.addLog("Tentando reconectar ao banco de dados...")

	conn, err := database.Connect()
	if err != nil {
		a.addLog(fmt.Sprintf("Falha na reconexão: %s", err.Error()))
		return err
	}

	if a.db != nil {
		a.db.Disconnect()
	}

	a.db = conn
	a.operations = operations.NewManager(conn)
	a.rollback = operations.NewRollbackManager(conn)
	a.operations.SetRollback(a.rollback)

	validator := database.NewValidator(conn)
	ok, msg := validator.ValidateConnection()
	if ok {
		a.addLog(fmt.Sprintf("%s", msg))

		empty, _ := validator.IsDatabaseEmpty()
		if empty {
			a.addLog("O banco de dados está vazio. Por favor, restaure uma base.")
		}
		return nil
	}

	a.addLog(fmt.Sprintf("%s", msg))
	return fmt.Errorf(msg)
}

func (a *App) InactivateZeroProducts() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	a.addLog("Buscando estoques zerados ou negativos...")

	count, err := a.operations.InactivateZeroProducts(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return 0, err
	}

	a.addLog(fmt.Sprintf("%d produtos inativados", count))
	return count, nil
}

func (a *App) ChangeTributationByNCM(ncms []string, tributationID string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	a.addLog(fmt.Sprintf("Alterando tributação para NCMs: %v", ncms))

	count, err := a.operations.ChangeTributationByNCM(ncms, tributationID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return 0, err
	}

	a.addLog(fmt.Sprintf("%d produtos atualizados", count))
	return count, nil
}

func (a *App) GetFederalTributations() []map[string]interface{} {
	if a.operations == nil {
		return []map[string]interface{}{}
	}
	res, err := a.operations.GetFederalTributations()
	if err != nil {
		a.addLog(fmt.Sprintf("Erro ao buscar tributações federais: %s", err.Error()))
		return []map[string]interface{}{}
	}
	return res
}

func (a *App) ChangeFederalTributationByNCM(ncms []string, tribID string) int {
	if a.operations == nil {
		return 0
	}

	a.addLog(fmt.Sprintf("Iniciando alteração de Tributação FEDERAL para NCMs: %v", ncms))
	err := a.operations.ChangeFederalTributationByNCM(ncms, tribID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return 0
	}
	return 1
}

func (a *App) GetTributations() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetTributations()
}

func (a *App) GetIbsCbsTributations() []map[string]interface{} {
	if a.operations == nil {
		return []map[string]interface{}{}
	}
	res, err := a.operations.GetIbsCbsTributations()
	if err != nil {
		a.addLog(fmt.Sprintf("Erro ao buscar tributações IBS/CBS: %s", err.Error()))
		return []map[string]interface{}{}
	}
	return res
}

func (a *App) ChangeIbsCbsTributationByNCM(ncms []string, tribID string) int {
	if a.operations == nil {
		return 0
	}

	a.addLog(fmt.Sprintf("Iniciando alteração de Tributação IBS/CBS para NCMs: %v", ncms))
	err := a.operations.ChangeIbsCbsTributationByNCM(ncms, tribID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return 0
	}
	return 1
}

func (a *App) EnableMEI() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	a.addLog("Habilitando ajuste de estoque MEI...")

	count, err := a.operations.EnableMEI(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return 0, err
	}

	a.addLog(fmt.Sprintf("%d referências alteradas", count))
	return count, nil
}

func (a *App) CleanMovements() error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog("Iniciando limpeza de movimentações...")

	err := a.operations.CleanMovements(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return err
	}

	a.addLog("Limpeza de movimentações concluída")
	return nil
}

func (a *App) FindObjectIdInDatabase(searchID string) ([]map[string]string, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	a.addLog(fmt.Sprintf("Buscando ObjectId %s em todas as coleções...", searchID))

	results, err := a.operations.FindObjectIdInDatabase(searchID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return nil, err
	}

	a.addLog(fmt.Sprintf("Busca concluída. Encontradas %d referências", len(results)))
	return results, nil
}

func (a *App) CleanDatabase() error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog("Iniciando limpeza da base de dados...")

	err := a.operations.CleanDatabase(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return err
	}

	a.addLog("Limpeza de base concluída!")
	return nil
}

func (a *App) CreateNewDatabase() error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog("Iniciando criação de nova base (ZERO)...")

	err := a.operations.CreateNewDatabase(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return err
	}

	a.addLog("Nova base criada com sucesso!")
	return nil
}

func (a *App) CleanDigisatRegistry() error {

	reg := windows.NewRegistryManager()

	err := reg.CleanDigisatRegistry(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro no registro: %s", err.Error()))
		return err
	}

	return nil
}

func (a *App) CancelOperation() {
	if a.operations != nil {
		a.operations.CancelAll()
		a.addLog("Operação cancelada")
	}
}

func (a *App) GetUndoableOperations() []map[string]interface{} {
	if a.rollback == nil {
		return []map[string]interface{}{}
	}

	ops := a.rollback.GetUndoableOperations()
	result := make([]map[string]interface{}, len(ops))
	for i, op := range ops {
		result[i] = map[string]interface{}{
			"id":        op.ID,
			"type":      string(op.Type),
			"timestamp": op.Timestamp.Format("15:04:05"),
			"label":     op.Label,
			"undoable":  op.Undoable,
		}
	}
	return result
}

func (a *App) UndoOperation(opID string) error {
	if a.rollback == nil {
		return fmt.Errorf("rollback não inicializado")
	}

	a.addLog(fmt.Sprintf("Revertendo operação %s...", opID))

	err := a.rollback.UndoOperation(opID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro no rollback: %s", err.Error()))
		return err
	}

	a.addLog("Operação revertida com sucesso!")
	return nil
}

func (a *App) FilterProducts(filter map[string]interface{}) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	pf := a.buildProductFilter(filter)

	results, err := a.operations.FilterProducts(pf, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

	products := make([]map[string]interface{}, len(results.Products))
	for i, r := range results.Products {
		products[i] = map[string]interface{}{
			"id":            r.ID,
			"empresaId":     r.EmpresaID,
			"name":          r.Name,
			"internalCode":  r.InternalCode,
			"barcode":       r.Barcode,
			"brand":         r.Brand,
			"brandId":       r.BrandID,
			"ncm":           r.NCM,
			"stateTribId":   r.StateTribID,
			"federalTribId": r.FederalTribID,
			"ibsCbsTribId":  r.IbsCbsTribID,
			"itemType":      r.ItemType,
			"itemTypeId":    r.ItemTypeID,
			"quantity":      r.Quantity,
			"costPrice":     r.CostPrice,
			"salePrice":     r.SalePrice,
			"active":        r.Active,
			"weighable":     r.Weighable,
		}
	}

	return map[string]interface{}{
		"products": products,
		"total":    results.Total,
		"limit":    results.Limit,
	}, nil
}

func (a *App) buildProductFilter(filter map[string]interface{}) operations.ProductFilter {
	pf := operations.ProductFilter{
		// Text filters
		Name:         getStringOrEmpty(filter, "name"),
		InternalCode: getStringOrEmpty(filter, "internalCode"),
		Barcode:      getStringOrEmpty(filter, "barcode"),
		Brand:        getStringOrEmpty(filter, "brand"),
		BrandID:      getStringOrEmpty(filter, "brandId"),

		// Tributation filters
		StateTribID:     getStringOrEmpty(filter, "stateTribId"),
		FederalTribID:   getStringOrEmpty(filter, "federalTribId"),
		MunicipalTribID: getStringOrEmpty(filter, "municipalTribId"),
		IbsCbsTribID:    getStringOrEmpty(filter, "ibsCbsTribId"),

		// Item type (fiscal)
		ItemType:   getStringOrEmpty(filter, "itemType"),
		ItemTypeID: getStringOrEmpty(filter, "itemTypeId"),

		// Product type (GeneroItem: Serviço, Produto, GLP, etc)
		ProductType:   getStringOrEmpty(filter, "productType"),
		ProductTypeID: getStringOrEmpty(filter, "productTypeId"),

		// Quantity
		QuantityOp:    getStringOrEmpty(filter, "quantityOp"),
		QuantityValue: getFloatOrZero(filter, "quantityValue"),

		// Prices
		CostPriceOp:  getStringOrEmpty(filter, "costPriceOp"),
		CostPriceVal: getFloatOrZero(filter, "costPriceVal"),
		SalePriceOp:  getStringOrEmpty(filter, "salePriceOp"),
		SalePriceVal: getFloatOrZero(filter, "salePriceVal"),
	}

	// NCMs array
	if ncmsRaw, ok := filter["ncms"].([]interface{}); ok {
		ncms := make([]string, len(ncmsRaw))
		for i, n := range ncmsRaw {
			ncms[i], _ = n.(string)
		}
		pf.NCMs = ncms
	}

	// Boolean filters
	if v, ok := filter["weighable"].(bool); ok {
		pf.Weighable = &v
	}
	if v, ok := filter["activeStatus"].(bool); ok {
		pf.ActiveStatus = &v
	}
	if v, ok := filter["hasCodigoTribMunicipio"].(bool); ok {
		pf.HasCodigoTribMunicipio = &v
	}

	return pf
}

func (a *App) CountFilteredProducts(filter map[string]interface{}) (int64, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	pf := a.buildProductFilter(filter)
	return a.operations.CountFilteredProducts(pf)
}

func (a *App) GetAllFilteredProductIDs(filter map[string]interface{}) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	pf := a.buildProductFilter(filter)
	productIDs, empresaIDs, err := a.operations.GetAllFilteredProductIDs(pf, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"productIds": productIDs,
		"empresaIds": empresaIDs,
	}, nil
}

func (a *App) ExecuteBulkOperation(opType string, productIDs []string, empresaIDs []string, useFilter bool, filter map[string]interface{}, operationValue interface{}) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	req := operations.BulkOperationRequest{
		ProductIDs:     productIDs,
		EmpresaIDs:     empresaIDs,
		UseFilter:      useFilter,
		Operation:      operations.BulkOperation(opType),
		OperationValue: operationValue,
	}

	if useFilter {
		req.Filter = a.buildProductFilter(filter)
	}

	a.addLog(fmt.Sprintf("🚀 Executando operação em massa: %s", opType))

	result, err := a.operations.ExecuteBulkOperation(req, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return nil, err
	}

	return map[string]interface{}{
		"totalAffected": result.TotalAffected,
		"message":       result.Message,
		"canRollback":   result.CanRollback,
		"operationId":   result.OperationID,
	}, nil
}

func (a *App) GetBrands() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetBrands()
}

func (a *App) GetItemTypes() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetItemTypes()
}

func (a *App) GetProductTypes() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetProductTypes()
}

func (a *App) GetMunicipalTributations() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetMunicipalTributations()
}

func (a *App) BulkActivateProducts(productIDs []string, activate bool) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.BulkActivateProducts(productIDs, activate, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) BulkActivateByFilter(filter map[string]interface{}, activate bool) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	pf := operations.ProductFilter{
		Brand:         getStringOrEmpty(filter, "brand"),
		StateTribID:   getStringOrEmpty(filter, "stateTribId"),
		FederalTribID: getStringOrEmpty(filter, "federalTribId"),
		ItemType:      getStringOrEmpty(filter, "itemType"),
	}

	if ncmsRaw, ok := filter["ncms"].([]interface{}); ok {
		ncms := make([]string, len(ncmsRaw))
		for i, n := range ncmsRaw {
			ncms[i], _ = n.(string)
		}
		pf.NCMs = ncms
	}

	if v, ok := filter["weighable"].(bool); ok {
		pf.Weighable = &v
	}
	if v, ok := filter["activeStatus"].(bool); ok {
		pf.ActiveStatus = &v
	}

	return a.operations.BulkActivateByFilter(pf, activate, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) ZeroAllStock() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.ZeroAllStock(func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) ZeroNegativeStock() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.ZeroNegativeStock(func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) ZeroAllPrices() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.ZeroAllPrices(func(msg string) {
		a.addLog(msg)
	})
}

// === Phase 1: Price Operations ===

func (a *App) AdjustPricesByPercent(filterParams map[string]interface{}, percent float64, priceType string) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	filter := a.buildPriceFilter(filterParams)
	pt := operations.PriceType(priceType)

	a.addLog(fmt.Sprintf("💰 Ajustando preços em %.2f%% (tipo: %s)...", percent, priceType))

	result, err := a.operations.AdjustPricesByPercent(filter, percent, pt, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalAffected": result.TotalAffected,
		"totalProducts": result.TotalProducts,
		"averageChange": result.AverageChange,
	}, nil
}

func (a *App) PreviewPriceAdjustment(filterParams map[string]interface{}, percent float64, priceType string, limit int) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	filter := a.buildPriceFilter(filterParams)
	pt := operations.PriceType(priceType)

	previews, total, err := a.operations.PreviewPriceAdjustment(filter, percent, pt, limit)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]interface{}, len(previews))
	for i, p := range previews {
		items[i] = map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"currentCost": p.CurrentCost,
			"currentSale": p.CurrentSale,
			"newCost":     p.NewCost,
			"newSale":     p.NewSale,
			"costChange":  p.CostChange,
			"saleChange":  p.SaleChange,
		}
	}

	return map[string]interface{}{
		"items": items,
		"total": total,
	}, nil
}

func (a *App) ApplyMarkup(filterParams map[string]interface{}, markupPercent float64) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	filter := a.buildPriceFilter(filterParams)

	a.addLog(fmt.Sprintf("💰 Aplicando markup de %.2f%%...", markupPercent))

	result, err := a.operations.ApplyMarkup(filter, markupPercent, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalAffected": result.TotalAffected,
		"averageChange": result.AverageChange,
	}, nil
}

func (a *App) ZeroPricesByFilter(filterParams map[string]interface{}, priceType string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	filter := a.buildPriceFilter(filterParams)
	pt := operations.PriceType(priceType)

	a.addLog(fmt.Sprintf("🔄 Zerando preços (%s) por filtro...", priceType))

	return a.operations.ZeroPricesByFilter(filter, pt, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) buildPriceFilter(params map[string]interface{}) operations.PriceFilter {
	filter := operations.PriceFilter{}

	if ncmsRaw, ok := params["ncms"].([]interface{}); ok {
		ncms := make([]string, len(ncmsRaw))
		for i, n := range ncmsRaw {
			ncms[i], _ = n.(string)
		}
		filter.NCMs = ncms
	}

	if brand, ok := params["brand"].(string); ok {
		filter.Brand = brand
	}

	if activeOnly, ok := params["activeOnly"].(bool); ok {
		filter.ActiveOnly = activeOnly
	}

	if qOp, ok := params["quantityOp"].(string); ok {
		filter.QuantityOp = qOp
	}

	if qVal, ok := params["quantityValue"].(float64); ok {
		filter.QuantityValue = qVal
	}

	return filter
}

// === Phase 1: NCM Operations ===

func (a *App) ChangeNCMByFilter(oldNCMPrefix string, newNCM string) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	a.addLog(fmt.Sprintf("🔄 Alterando NCM: %s → %s...", oldNCMPrefix, newNCM))

	result, err := a.operations.ChangeNCMByFilter(oldNCMPrefix, newNCM, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalAffected": result.TotalAffected,
		"oldNCM":        result.OldNCM,
		"newNCM":        result.NewNCM,
	}, nil
}

func (a *App) PreviewNCMChange(oldNCMPrefix string, newNCM string, limit int) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	previews, total, err := a.operations.PreviewNCMChange(oldNCMPrefix, newNCM, limit)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]interface{}, len(previews))
	for i, p := range previews {
		items[i] = map[string]interface{}{
			"id":         p.ID,
			"name":       p.Name,
			"currentNCM": p.CurrentNCM,
			"newNCM":     p.NewNCM,
		}
	}

	return map[string]interface{}{
		"items": items,
		"total": total,
	}, nil
}

func (a *App) GetInvalidNCMs(limit int) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	items, total, err := a.operations.GetInvalidNCMs(limit)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		result[i] = map[string]interface{}{
			"id":         item.ID,
			"name":       item.Name,
			"currentNCM": item.CurrentNCM,
			"reason":     item.Reason,
		}
	}

	return map[string]interface{}{
		"items": result,
		"total": total,
	}, nil
}

func (a *App) GetDistinctNCMs() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.GetDistinctNCMs()
}

func (a *App) CleanDatabaseByDate(beforeDate string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.CleanDatabaseByDate(beforeDate, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) GetInventoryValue(cutoffDate string) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	result, err := a.operations.GetInventoryValue(func(msg string) {
		a.addLog(msg)
	}, cutoffDate)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"currentValue":   result.CurrentValue,
		"productCount":   result.ProductCount,
		"lowCostCount":   result.LowCostCount,
		"lowCostPercent": result.LowCostPercent,
		"message":        result.Message,
	}, nil
}

func (a *App) SanitizePrices(percent float64) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.SanitizePrices(percent, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) AdjustInventoryRebalance(targetValue float64, resetToZero bool, cutoffDate string) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	result, err := a.operations.AdjustInventoryRebalance(targetValue, resetToZero, func(msg string) {
		a.addLog(msg)
	}, cutoffDate)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"adjustedCount": result.AdjustedCount,
		"zeroedCount":   result.ZeroedCount,
		"previousValue": result.PreviousValue,
		"newValue":      result.NewValue,
		"targetValue":   result.TargetValue,
		"maxAdjustment": result.MaxAdjustment,
		"message":       result.Message,
	}, nil
}

func (a *App) GenerateInventoryReport(cutoffDate string, targetValue float64, format string, companyName string, companyIE string, companyCNPJ string, bookNumber int, sheetNumber int) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	format = strings.ToUpper(format)

	parsedDate, err := time.Parse("02/01/2006", cutoffDate)
	if err != nil {
		parsedDate, _ = time.Parse("2006-01-02", cutoffDate)
	}
	defaultName := fmt.Sprintf("Inventario_%s_%s", strings.ReplaceAll(companyName, " ", "_"), parsedDate.Format("20060102"))

	var filters []runtime.FileFilter
	if format == "PDF" {
		filters = []runtime.FileFilter{{DisplayName: "Arquivos PDF (*.pdf)", Pattern: "*.pdf"}}
		defaultName += ".pdf"
	} else if format == "CSV" {
		filters = []runtime.FileFilter{{DisplayName: "Arquivos CSV (*.csv)", Pattern: "*.csv"}}
		defaultName += ".csv"
	} else if format == "EXCEL" || format == "XLSX" {
		filters = []runtime.FileFilter{{DisplayName: "Arquivos Excel (*.xlsx)", Pattern: "*.xlsx"}}
		defaultName += ".xlsx"
	} else {
		filters = []runtime.FileFilter{
			{DisplayName: "Todos os Arquivos (*.*)", Pattern: "*.*"},
		}
	}

	// Select Output File (Save As)
	selectedPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvar Relatório Como...",
		DefaultFilename: defaultName,
		Filters:         filters,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao selecionar arquivo: %v", err)
	}
	if selectedPath == "" {
		return nil, fmt.Errorf("salvamento cancelado")
	}

	params := operations.InventoryReportParams{
		CutoffDate:  cutoffDate,
		TargetValue: targetValue,
		CompanyName: companyName,
		CompanyIE:   companyIE,
		CompanyCNPJ: companyCNPJ,
		BookNumber:  bookNumber,
		SheetNumber: sheetNumber,
	}

	result, err := a.operations.GenerateInventoryReport(params, selectedPath, format, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

	totalFloat, _ := result.TotalValue.Float64()
	return map[string]interface{}{
		"totalItems": result.TotalItems,
		"totalValue": totalFloat,
		"outputPath": result.OutputPath,
		"message":    result.Message,
	}, nil
}

func (a *App) GetTotalProductCount() (int64, error) {
	if a.db == nil {
		return 0, nil
	}
	ctx := context.Background()
	count, err := a.db.GetCollection(database.CollectionProdutosServicos).CountDocuments(ctx, map[string]interface{}{})
	return count, err
}

func (a *App) SelectInfoDatFile() (string, error) {
	options := runtime.OpenDialogOptions{
		Title: "Selecione o arquivo info.dat",
		Filters: []runtime.FileFilter{
			{DisplayName: "Arquivos DAT", Pattern: "*.dat"},
			{DisplayName: "Todos os arquivos", Pattern: "*.*"},
		},
		DefaultFilename: "info.dat",
	}
	filePath, err := runtime.OpenFileDialog(a.ctx, options)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func (a *App) UpdateEmitenteFromFile(filePath string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog(fmt.Sprintf("Lendo arquivo: %s", filePath))

	info, err := operations.ParseInfoDat(filePath)
	if err != nil {
		a.addLog(fmt.Sprintf("Erro ao ler arquivo: %s", err.Error()))
		return err
	}

	a.addLog(fmt.Sprintf("Dados lidos - CNPJ: %s, Razão: %s", info.Cnpj, info.RazaoSocial))

	err = a.operations.UpdateEmitente(info, filePath, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("Erro: %s", err.Error()))
		return err
	}

	return nil
}

func (a *App) ListEmitentes() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	emitentes, err := a.operations.ListEmitentes(func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(emitentes))
	for i, e := range emitentes {
		result[i] = map[string]interface{}{
			"id":                e.ID,
			"nome":              e.Nome,
			"cnpj":              e.Cnpj,
			"inscricaoEstadual": e.InscricaoEstadual,
		}
	}
	return result, nil
}

func (a *App) DeleteEmitente(emitenteID string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	return a.operations.DeleteEmitente(emitenteID, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) ChangeInvoiceKey(invoiceType string, oldKey string, newKey string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.ChangeInvoiceKey(invoiceType, oldKey, newKey, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) ChangeInvoiceStatus(invoiceType string, serie string, numero string, newStatus string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	return a.operations.ChangeInvoiceStatus(invoiceType, serie, numero, newStatus, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) GetInvoiceTypes() []string {
	return operations.GetInvoiceTypes()
}

func (a *App) GetInvoiceStatuses() []string {
	return operations.GetInvoiceStatuses()
}

func (a *App) GetInvoiceByKey(invoiceType string, key string) (*operations.InvoiceDetails, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.GetInvoiceByKey(invoiceType, key, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) GetInvoiceByNumber(invoiceType string, serie string, number string) (*operations.InvoiceDetails, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.GetInvoiceByNumber(invoiceType, serie, number, func(msg string) {
		a.addLog(msg)
	})
}

func getStringOrEmpty(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloatOrZero(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func (a *App) BackupDatabase(outputDir string) (*operations.BackupResult, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.BackupDatabase(outputDir, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) RestoreDatabase(backupPath string, dropExisting bool) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	return a.operations.RestoreDatabase(backupPath, dropExisting, func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) ListBackups(backupDir string) ([]operations.BackupResult, error) {
	return operations.ListBackups(backupDir)
}

func (a *App) GetDigisatServices() ([]windows.DigiService, error) {
	return windows.GetDigisatServices()
}

func (a *App) StopDigisatServices() (int, error) {
	return windows.StopDigisatServices(func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) StartDigisatServices() (int, error) {
	return windows.StartDigisatServices(func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) KillDigisatProcesses() (int, error) {
	return windows.KillDigisatProcesses(func(msg string) {
		a.addLog(msg)
	})
}

func (a *App) GetDigisatProcesses() ([]windows.DigiProcess, error) {
	return windows.GetDigisatProcesses()
}

func (a *App) SelectDirectory(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

func (a *App) SelectBackupFile(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "Backup ZIP (*.zip)", Pattern: "*.zip"},
			{DisplayName: "Todos os arquivos (*.*)", Pattern: "*.*"},
		},
	})
}

func (a *App) RepairMongoDBOffline() error {
	a.addLog("🔧 Iniciando reparo offline do MongoDB...")

	err := windows.RepairMongoDB(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro no reparo: %s", err.Error()))
		return err
	}

	return nil
}

func (a *App) RepairMongoDBOnline() error {
	a.addLog("🔧 Iniciando reparo online do MongoDB...")

	mongoURL := fmt.Sprintf("mongodb://%s:%s@%s:12220/DigisatServer?authSource=admin",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
	)

	err := windows.RepairMongoDBActive(mongoURL, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro no reparo: %s", err.Error()))
		return err
	}

	return nil
}

func (a *App) ReleaseFirewallPorts() error {
	a.addLog("🔥 Liberando portas no firewall...")

	err := windows.ReleaseFirewallPorts(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return err
	}

	return nil
}

func (a *App) AllowSecurityExclusions() error {
	a.addLog("🛡️ Configurando exclusões de segurança...")

	err := windows.AllowSecurityExclusions(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return err
	}

	return nil
}

// === Manual Invoices ===

func (a *App) GetManualInvoices(limit int) ([]operations.InvoiceSummary, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetManualInvoices(limit)
}

func (a *App) GetInvoiceData(invoiceID string) (*operations.InvoiceData, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetInvoiceData(invoiceID)
}

// GetSuggestedInvoiceNumber returns the suggested number for an invoice
func (a *App) GetSuggestedInvoiceNumber(invoiceID string) int64 {
	return a.numberManager.GetSuggestedNumber(invoiceID)
}

// ConfirmInvoiceNumber confirms the number used for an invoice
func (a *App) ConfirmInvoiceNumber(invoiceID string, number int64) {
	a.numberManager.ConfirmNumber(invoiceID, number)
}

// PrintInvoiceToBrowser generates a standalone HTML and opens it in default browser
func (a *App) PrintInvoiceToBrowser(invoiceID, manualNumber, batch string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	data, err := a.operations.GetInvoiceData(invoiceID)
	if err != nil {
		return err
	}

	filePath, err := a.operations.GenerateInvoiceHTML(data, manualNumber, batch)
	if err != nil {
		return err
	}

	runtime.BrowserOpenURL(a.ctx, "file://"+filePath)
	return nil
}

// ExportInvoiceToPDF generates a direct PDF and opens it
func (a *App) ExportInvoiceToPDF(invoiceID, manualNumber, batch string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	data, err := a.operations.GetInvoiceData(invoiceID)
	if err != nil {
		return err
	}

	// 1. Ask for save path
	defaultName := fmt.Sprintf("Fatura_%s.pdf", manualNumber)
	if manualNumber == "" {
		defaultName = fmt.Sprintf("Fatura_%d.pdf", data.Numero)
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvar Fatura como PDF",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Arquivos PDF (*.pdf)", Pattern: "*.pdf"},
		},
	})

	if err != nil {
		return err
	}

	if savePath == "" {
		return nil // User cancelled
	}

	// 2. Generate PDF at chosen path
	err = a.operations.GenerateInvoicePDF(data, manualNumber, batch, savePath)
	if err != nil {
		return err
	}

	// 3. Open the generated file
	// Correct file URL format for Windows: file:///C:/path/to/file
	fileURL := "file:///" + filepath.ToSlash(savePath)
	runtime.BrowserOpenURL(a.ctx, fileURL)
	return nil
}

package main

import (
	"context"
	"fmt"

	"BMongo-VIP/internal/database"
	"BMongo-VIP/internal/operations"
	"BMongo-VIP/internal/windows"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct represents the main application
type App struct {
	ctx        context.Context
	db         *database.Connection
	operations *operations.Manager
	rollback   *operations.RollbackManager
	logs       []string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		logs: make([]string, 0),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Connect to database
	conn, err := database.Connect()
	if err != nil {
		a.addLog(fmt.Sprintf("⚠️ Erro: %s", err.Error()))
		return
	}

	a.db = conn
	a.operations = operations.NewManager(conn)
	a.rollback = operations.NewRollbackManager(conn)

	// Validate connection
	validator := database.NewValidator(conn)
	ok, msg := validator.ValidateConnection()
	if ok {
		a.addLog(fmt.Sprintf("✅ %s", msg))

		// Check if database is empty
		empty, _ := validator.IsDatabaseEmpty()
		if empty {
			a.addLog("⚠️ O banco de dados está vazio. Por favor, restaure uma base.")
		}
	} else {
		a.addLog(fmt.Sprintf("❌ %s", msg))
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Disconnect()
	}
}

// addLog adds a log message and emits it to the frontend
func (a *App) addLog(message string) {
	a.logs = append(a.logs, message)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "log", message)
	}
}

// GetLogs returns all log messages
func (a *App) GetLogs() []string {
	return a.logs
}

// ClearLogs clears all log messages
func (a *App) ClearLogs() {
	a.logs = make([]string, 0)
}

// CheckConnection checks if database is connected
func (a *App) CheckConnection() bool {
	if a.db == nil {
		return false
	}
	return a.db.IsConnected()
}

// ============================================
// Product Operations
// ============================================

// InactivateZeroProducts deactivates products with zero or negative stock
func (a *App) InactivateZeroProducts() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	a.addLog("🔄 Buscando estoques zerados ou negativos...")

	count, err := a.operations.InactivateZeroProducts(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return 0, err
	}

	a.addLog(fmt.Sprintf("✅ %d produtos inativados", count))
	return count, nil
}

// ChangeTributationByNCM changes tributation for products by NCM
func (a *App) ChangeTributationByNCM(ncms []string, tributationID string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	a.addLog(fmt.Sprintf("🔄 Alterando tributação para NCMs: %v", ncms))

	count, err := a.operations.ChangeTributationByNCM(ncms, tributationID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return 0, err
	}

	a.addLog(fmt.Sprintf("✅ %d produtos atualizados", count))
	return count, nil
}

// GetFederalTributations returns all active federal tributations
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

// ChangeFederalTributationByNCM receives a list of NCMs and a Tributation ID to update products
func (a *App) ChangeFederalTributationByNCM(ncms []string, tribID string) int {
	if a.operations == nil {
		return 0
	}

	a.addLog(fmt.Sprintf("🔄 Iniciando alteração de Tributação FEDERAL para NCMs: %v", ncms))
	err := a.operations.ChangeFederalTributationByNCM(ncms, tribID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return 0
	}
	return 1
}

// GetTributations returns active tributations
func (a *App) GetTributations() ([]map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.GetTributations()
}

// EnableMEI enables MEI adjustment for stock
func (a *App) EnableMEI() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	a.addLog("🔄 Habilitando ajuste de estoque MEI...")

	count, err := a.operations.EnableMEI(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return 0, err
	}

	a.addLog(fmt.Sprintf("✅ %d referências alteradas", count))
	return count, nil
}

// ============================================
// Movement Operations
// ============================================

// CleanMovements cleans movement data from the database
func (a *App) CleanMovements() error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog("🔄 Iniciando limpeza de movimentações...")

	err := a.operations.CleanMovements(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return err
	}

	a.addLog("✅ Limpeza de movimentações concluída")
	return nil
}

// ============================================
// Search Operations
// ============================================

// FindObjectIdInDatabase searches for an ObjectId in all collections
func (a *App) FindObjectIdInDatabase(searchID string) ([]map[string]string, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	a.addLog(fmt.Sprintf("🔍 Buscando ObjectId %s em todas as coleções...", searchID))

	results, err := a.operations.FindObjectIdInDatabase(searchID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return nil, err
	}

	a.addLog(fmt.Sprintf("✅ Busca concluída. Encontradas %d referências", len(results)))
	return results, nil
}

// ============================================
// Base Operations
// ============================================

// CleanDatabase cleans the database keeping only essential collections
func (a *App) CleanDatabase() error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog("🧹 Iniciando limpeza da base de dados...")

	err := a.operations.CleanDatabase(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return err
	}

	a.addLog("✅ Limpeza de base concluída!")
	return nil
}

// CreateNewDatabase drops everything to create a fresh start
func (a *App) CreateNewDatabase() error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog("🗑️ Iniciando criação de nova base (ZERO)...")

	err := a.operations.CreateNewDatabase(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return err
	}

	a.addLog("✅ Nova base criada com sucesso!")
	return nil
}

// ============================================
// Windows Operations
// ============================================

// CleanDigisatRegistry cleans Digisat registry keys
func (a *App) CleanDigisatRegistry() error {
	// Import needs to be added to file
	reg := windows.NewRegistryManager()

	err := reg.CleanDigisatRegistry(func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro no registro: %s", err.Error()))
		return err
	}

	return nil
}

// ============================================
// Cancel Operation
// ============================================

// CancelOperation cancels all running operations
func (a *App) CancelOperation() {
	if a.operations != nil {
		a.operations.CancelAll()
		a.addLog("⏹️ Operação cancelada")
	}
}

// ============================================
// Rollback Operations
// ============================================

// GetUndoableOperations returns list of operations that can be undone
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

// UndoOperation reverts a specific operation
func (a *App) UndoOperation(opID string) error {
	if a.rollback == nil {
		return fmt.Errorf("rollback não inicializado")
	}

	a.addLog(fmt.Sprintf("🔄 Revertendo operação %s...", opID))

	err := a.rollback.UndoOperation(opID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro no rollback: %s", err.Error()))
		return err
	}

	a.addLog("✅ Operação revertida com sucesso!")
	return nil
}

// ============================================
// Product Filter Operations
// ============================================

// FilterProducts searches products based on filter criteria
func (a *App) FilterProducts(filter map[string]interface{}) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	// Convert map to ProductFilter struct
	pf := operations.ProductFilter{
		QuantityOp:    getStringOrEmpty(filter, "quantityOp"),
		QuantityValue: getFloatOrZero(filter, "quantityValue"),
		Brand:         getStringOrEmpty(filter, "brand"),
		StateTribID:   getStringOrEmpty(filter, "stateTribId"),
		FederalTribID: getStringOrEmpty(filter, "federalTribId"),
		ItemType:      getStringOrEmpty(filter, "itemType"),
		CostPriceOp:   getStringOrEmpty(filter, "costPriceOp"),
		CostPriceVal:  getFloatOrZero(filter, "costPriceVal"),
		SalePriceOp:   getStringOrEmpty(filter, "salePriceOp"),
		SalePriceVal:  getFloatOrZero(filter, "salePriceVal"),
	}

	// NCMs
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

	results, err := a.operations.FilterProducts(pf, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

	// Convert to []map for frontend
	products := make([]map[string]interface{}, len(results.Products))
	for i, r := range results.Products {
		products[i] = map[string]interface{}{
			"id":        r.ID,
			"name":      r.Name,
			"brand":     r.Brand,
			"ncm":       r.NCM,
			"quantity":  r.Quantity,
			"costPrice": r.CostPrice,
			"salePrice": r.SalePrice,
			"active":    r.Active,
			"weighable": r.Weighable,
			"itemType":  r.ItemType,
		}
	}

	// Return with metadata
	return map[string]interface{}{
		"products": products,
		"total":    results.Total,
		"limit":    results.Limit,
	}, nil
}

// BulkActivateProducts activates or deactivates products in bulk
func (a *App) BulkActivateProducts(productIDs []string, activate bool) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.BulkActivateProducts(productIDs, activate, func(msg string) {
		a.addLog(msg)
	})
}

// BulkActivateByFilter activates/deactivates ALL products matching filter
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

// ============================================
// Stock & Price Operations (Fase 4)
// ============================================

// ZeroAllStock sets all product stock quantities to zero
func (a *App) ZeroAllStock() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.ZeroAllStock(func(msg string) {
		a.addLog(msg)
	})
}

// ZeroNegativeStock sets only negative stock quantities to zero
func (a *App) ZeroNegativeStock() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.ZeroNegativeStock(func(msg string) {
		a.addLog(msg)
	})
}

// ZeroAllPrices sets all product prices to zero
func (a *App) ZeroAllPrices() (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.ZeroAllPrices(func(msg string) {
		a.addLog(msg)
	})
}

// CleanDatabaseByDate removes movements before specified date
func (a *App) CleanDatabaseByDate(beforeDate string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.CleanDatabaseByDate(beforeDate, func(msg string) {
		a.addLog(msg)
	})
}

// GetTotalProductCount returns total count of products in database
func (a *App) GetTotalProductCount() (int64, error) {
	if a.db == nil {
		return 0, nil
	}
	ctx := context.Background()
	count, err := a.db.GetCollection(database.CollectionProdutosServicos).CountDocuments(ctx, map[string]interface{}{})
	return count, err
}

// ============================================
// Emitente Operations (Fase 3)
// ============================================

// SelectInfoDatFile opens native file dialog and returns selected path
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

// UpdateEmitenteFromFile reads info.dat and updates Matriz
func (a *App) UpdateEmitenteFromFile(filePath string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	a.addLog(fmt.Sprintf("📂 Lendo arquivo: %s", filePath))

	info, err := operations.ParseInfoDat(filePath)
	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro ao ler arquivo: %s", err.Error()))
		return err
	}

	a.addLog(fmt.Sprintf("📋 Dados lidos - CNPJ: %s, Razão: %s", info.Cnpj, info.RazaoSocial))

	err = a.operations.UpdateEmitente(info, filePath, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return err
	}

	return nil
}

// ListEmitentes returns all emitentes in the database
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

	// Convert to map for frontend
	result := make([]map[string]interface{}, len(emitentes))
	for i, e := range emitentes {
		result[i] = map[string]interface{}{
			"id":   e.ID,
			"nome": e.Nome,
			"cnpj": e.Cnpj,
		}
	}
	return result, nil
}

// DeleteEmitente removes an emitente
func (a *App) DeleteEmitente(emitenteID string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	return a.operations.DeleteEmitente(emitenteID, func(msg string) {
		a.addLog(msg)
	})
}

// ============================================
// Invoice Operations (Fase 5)
// ============================================

// ChangeInvoiceKey updates an invoice's access key
func (a *App) ChangeInvoiceKey(invoiceType string, oldKey string, newKey string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.ChangeInvoiceKey(invoiceType, oldKey, newKey, func(msg string) {
		a.addLog(msg)
	})
}

// ChangeInvoiceStatus updates an invoice's status
func (a *App) ChangeInvoiceStatus(invoiceType string, serie string, numero string, newStatus string) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	return a.operations.ChangeInvoiceStatus(invoiceType, serie, numero, newStatus, func(msg string) {
		a.addLog(msg)
	})
}

// GetInvoiceTypes returns available invoice types
func (a *App) GetInvoiceTypes() []string {
	return operations.GetInvoiceTypes()
}

// GetInvoiceStatuses returns available invoice statuses
func (a *App) GetInvoiceStatuses() []string {
	return operations.GetInvoiceStatuses()
}

// GetInvoiceByKey finds an invoice by its key
func (a *App) GetInvoiceByKey(invoiceType string, key string) (*operations.InvoiceDetails, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.GetInvoiceByKey(invoiceType, key, func(msg string) {
		a.addLog(msg)
	})
}

// GetInvoiceByNumber finds an invoice by number (and series)
func (a *App) GetInvoiceByNumber(invoiceType string, serie string, number string) (*operations.InvoiceDetails, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.GetInvoiceByNumber(invoiceType, serie, number, func(msg string) {
		a.addLog(msg)
	})
}

// Helper functions for map conversion
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

// ============================================
// Backup Operations
// ============================================

// BackupDatabase creates a backup of the database
func (a *App) BackupDatabase(outputDir string) (*operations.BackupResult, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

	return a.operations.BackupDatabase(outputDir, func(msg string) {
		a.addLog(msg)
	})
}

// RestoreDatabase restores the database from a backup
func (a *App) RestoreDatabase(backupPath string, dropExisting bool) error {
	if a.operations == nil {
		return fmt.Errorf("operações não inicializadas")
	}

	return a.operations.RestoreDatabase(backupPath, dropExisting, func(msg string) {
		a.addLog(msg)
	})
}

// ListBackups returns available backups in a directory
func (a *App) ListBackups(backupDir string) ([]operations.BackupResult, error) {
	return operations.ListBackups(backupDir)
}

// ============================================
// Windows Service Operations
// ============================================

// GetDigisatServices returns list of Digisat services
func (a *App) GetDigisatServices() ([]windows.DigiService, error) {
	return windows.GetDigisatServices()
}

// StopDigisatServices stops all Digisat services
func (a *App) StopDigisatServices() (int, error) {
	return windows.StopDigisatServices(func(msg string) {
		a.addLog(msg)
	})
}

// StartDigisatServices starts all Digisat services
func (a *App) StartDigisatServices() (int, error) {
	return windows.StartDigisatServices(func(msg string) {
		a.addLog(msg)
	})
}

// KillDigisatProcesses forcefully kills all Digisat processes
func (a *App) KillDigisatProcesses() (int, error) {
	return windows.KillDigisatProcesses(func(msg string) {
		a.addLog(msg)
	})
}

// GetDigisatProcesses returns list of running Digisat processes
func (a *App) GetDigisatProcesses() ([]windows.DigiProcess, error) {
	return windows.GetDigisatProcesses()
}

// SelectDirectory opens a folder picker dialog
func (a *App) SelectDirectory(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

// SelectBackupFile opens a file picker for backup files (ZIP or folder)
func (a *App) SelectBackupFile(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "Backup ZIP (*.zip)", Pattern: "*.zip"},
			{DisplayName: "Todos os arquivos (*.*)", Pattern: "*.*"},
		},
	})
}

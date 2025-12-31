package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"

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
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	conn, err := database.Connect()
	if err != nil {
		a.addLog(fmt.Sprintf("⚠️ Erro: %s", err.Error()))
		return
	}

	a.db = conn
	a.operations = operations.NewManager(conn)
	a.rollback = operations.NewRollbackManager(conn)

	validator := database.NewValidator(conn)
	ok, msg := validator.ValidateConnection()
	if ok {
		a.addLog(fmt.Sprintf("✅ %s", msg))

		empty, _ := validator.IsDatabaseEmpty()
		if empty {
			a.addLog("⚠️ O banco de dados está vazio. Por favor, restaure uma base.")
		}
	} else {
		a.addLog(fmt.Sprintf("❌ %s", msg))
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
	a.addLog("🔄 Tentando reconectar ao banco de dados...")

	conn, err := database.Connect()
	if err != nil {
		a.addLog(fmt.Sprintf("❌ Falha na reconexão: %s", err.Error()))
		return err
	}

	if a.db != nil {
		a.db.Disconnect()
	}

	a.db = conn
	a.operations = operations.NewManager(conn)
	a.rollback = operations.NewRollbackManager(conn)

	validator := database.NewValidator(conn)
	ok, msg := validator.ValidateConnection()
	if ok {
		a.addLog(fmt.Sprintf("✅ %s", msg))

		empty, _ := validator.IsDatabaseEmpty()
		if empty {
			a.addLog("⚠️ O banco de dados está vazio. Por favor, restaure uma base.")
		}
		return nil
	}

	a.addLog(fmt.Sprintf("❌ %s", msg))
	return fmt.Errorf(msg)
}

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

	a.addLog(fmt.Sprintf("🔄 Iniciando alteração de Tributação IBS/CBS para NCMs: %v", ncms))
	err := a.operations.ChangeIbsCbsTributationByNCM(ncms, tribID, func(msg string) {
		a.addLog(msg)
	})

	if err != nil {
		a.addLog(fmt.Sprintf("❌ Erro: %s", err.Error()))
		return 0
	}
	return 1
}

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

func (a *App) CleanDigisatRegistry() error {

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

func (a *App) CancelOperation() {
	if a.operations != nil {
		a.operations.CancelAll()
		a.addLog("⏹️ Operação cancelada")
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

func (a *App) FilterProducts(filter map[string]interface{}) (map[string]interface{}, error) {
	if a.operations == nil {
		return nil, fmt.Errorf("operações não inicializadas")
	}

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

	results, err := a.operations.FilterProducts(pf, func(msg string) {
		a.addLog(msg)
	})
	if err != nil {
		return nil, err
	}

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

	return map[string]interface{}{
		"products": products,
		"total":    results.Total,
		"limit":    results.Limit,
	}, nil
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

func (a *App) CleanDatabaseByDate(beforeDate string) (int, error) {
	if a.operations == nil {
		return 0, fmt.Errorf("operações não inicializadas")
	}
	return a.operations.CleanDatabaseByDate(beforeDate, func(msg string) {
		a.addLog(msg)
	})
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
			"id":   e.ID,
			"nome": e.Nome,
			"cnpj": e.Cnpj,
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

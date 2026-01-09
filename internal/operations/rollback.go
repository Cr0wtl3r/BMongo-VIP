package operations

import (
	"BMongo-VIP/internal/database"
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OperationType string

const (
	OpInactivateProducts  OperationType = "InactivateProducts"
	OpChangeTributation   OperationType = "ChangeTributation"
	OpChangeTribFederal   OperationType = "ChangeTributationFederal"
	OpChangeTribIbsCbs    OperationType = "ChangeTributationIbsCbs"
	OpBulkActivate        OperationType = "BulkActivate"
	OpZeroStock           OperationType = "ZeroStock"
	OpZeroNegativeStock   OperationType = "ZeroNegativeStock"
	OpZeroAllPrices       OperationType = "ZeroAllPrices"
	OpChangeInvoiceKey    OperationType = "ChangeInvoiceKey"
	OpChangeInvoiceStatus OperationType = "ChangeInvoiceStatus"
	OpEnableMEI           OperationType = "EnableMEI"
	OpAdjustInventory     OperationType = "AdjustInventory"
)

type OperationRecord struct {
	ID        string                 `json:"id"`
	Type      OperationType          `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Label     string                 `json:"label"`
	Details   map[string]interface{} `json:"details"`
	Undoable  bool                   `json:"undoable"`
}

type RollbackManager struct {
	history []OperationRecord
	mu      sync.RWMutex
	conn    *database.Connection
	maxOps  int
}

func NewRollbackManager(conn *database.Connection) *RollbackManager {
	return &RollbackManager{
		history: make([]OperationRecord, 0),
		conn:    conn,
		maxOps:  20,
	}
}

func (rm *RollbackManager) RecordOperation(opType OperationType, label string, details map[string]interface{}, undoable bool) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	id := primitive.NewObjectID().Hex()
	record := OperationRecord{
		ID:        id,
		Type:      opType,
		Timestamp: time.Now(),
		Label:     label,
		Details:   details,
		Undoable:  undoable,
	}

	rm.history = append(rm.history, record)

	if len(rm.history) > rm.maxOps {
		rm.history = rm.history[len(rm.history)-rm.maxOps:]
	}

	return id
}

func (rm *RollbackManager) GetUndoableOperations() []OperationRecord {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	undoable := make([]OperationRecord, 0)
	for i := len(rm.history) - 1; i >= 0; i-- {
		if rm.history[i].Undoable {
			undoable = append(undoable, rm.history[i])
		}
	}
	return undoable
}

func (rm *RollbackManager) UndoOperation(opID string, log LogFunc) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var target *OperationRecord
	var targetIdx int
	for i, op := range rm.history {
		if op.ID == opID {
			target = &rm.history[i]
			targetIdx = i
			break
		}
	}

	if target == nil {
		return fmt.Errorf("operação não encontrada: %s", opID)
	}

	if !target.Undoable {
		return fmt.Errorf("esta operação não pode ser revertida")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var err error
	switch target.Type {
	case OpInactivateProducts:
		err = rm.undoInactivateProducts(ctx, target.Details, log)
	case OpChangeTributation, OpChangeTribFederal:
		err = rm.undoTributationChange(ctx, target.Details, log)
	case OpChangeTribIbsCbs:
		err = rm.undoIbsCbsTributationChange(ctx, target.Details, log)
	case OpBulkActivate:
		err = rm.undoBulkActivate(ctx, target.Details, log)
	case OpZeroStock, OpZeroNegativeStock:
		err = rm.undoZeroStock(ctx, target.Details, log)
	case OpZeroAllPrices:
		err = rm.undoZeroAllPrices(ctx, target.Details, log)
	case OpChangeInvoiceKey:
		err = rm.undoInvoiceKeyChange(ctx, target.Details, log)
	case OpChangeInvoiceStatus:
		err = rm.undoInvoiceStatusChange(ctx, target.Details, log)
	case OpEnableMEI:
		err = rm.undoEnableMEI(ctx, target.Details, log)
	case OpAdjustInventory:
		err = rm.undoAdjustInventory(ctx, target.Details, log)
	default:
		return fmt.Errorf("tipo de operação não suportado para rollback: %s", target.Type)
	}

	if err != nil {
		return err
	}

	rm.history = append(rm.history[:targetIdx], rm.history[targetIdx+1:]...)
	return nil
}

func (rm *RollbackManager) undoInactivateProducts(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	productIDs, ok := details["productIds"].([]string)
	if !ok || len(productIDs) == 0 {
		if rawIDs, ok := details["productIds"].([]interface{}); ok {
			productIDs = make([]string, len(rawIDs))
			for i, id := range rawIDs {
				productIDs[i], _ = id.(string)
			}
		}
	}

	if len(productIDs) == 0 {
		return fmt.Errorf("nenhum produto para reativar")
	}

	log(fmt.Sprintf("🔄 Reativando %d produtos...", len(productIDs)))

	produtosServicos := rm.conn.GetCollection(database.CollectionProdutosServicos)
	count := 0

	for _, idHex := range productIDs {
		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			continue
		}

		result, err := produtosServicos.UpdateOne(ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{"Ativo": true}},
		)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ %d produtos reativados", count))
	return nil
}

func (rm *RollbackManager) undoTributationChange(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	changes, ok := details["changes"].([]map[string]interface{})
	if !ok || len(changes) == 0 {
		if rawChanges, ok := details["changes"].([]interface{}); ok {
			changes = make([]map[string]interface{}, len(rawChanges))
			for i, c := range rawChanges {
				changes[i], _ = c.(map[string]interface{})
			}
		}
	}

	if len(changes) == 0 {
		return fmt.Errorf("nenhuma alteração para reverter")
	}

	log(fmt.Sprintf("🔄 Revertendo tributação de %d produtos...", len(changes)))

	produtosEmpresa := rm.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	count := 0

	for _, change := range changes {
		productIDHex, _ := change["productId"].(string)
		prevTribHex, _ := change["prevTribId"].(string)
		isFederal, _ := change["isFederal"].(bool)

		if productIDHex == "" {
			continue
		}

		productOID, err := primitive.ObjectIDFromHex(productIDHex)
		if err != nil {
			continue
		}

		var update bson.M
		if isFederal {
			if prevTribHex != "" {
				tribOID, _ := primitive.ObjectIDFromHex(prevTribHex)
				update = bson.M{"$set": bson.M{"TributacaoFederalReferencia": tribOID}}
			} else {
				update = bson.M{"$unset": bson.M{"TributacaoFederalReferencia": 1, "TributacaoFederal": 1}}
			}
		} else {
			if prevTribHex != "" {
				tribOID, _ := primitive.ObjectIDFromHex(prevTribHex)
				update = bson.M{"$set": bson.M{"TributacaoEstadualReferencia": tribOID}}
			} else {
				update = bson.M{"$unset": bson.M{"TributacaoEstadualReferencia": 1}}
			}
		}

		result, err := produtosEmpresa.UpdateOne(ctx, bson.M{"_id": productOID}, update)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ Tributação revertida para %d produtos", count))
	return nil
}

func (rm *RollbackManager) undoIbsCbsTributationChange(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	changes, ok := details["changes"].([]map[string]interface{})
	if !ok {
		if rawChanges, ok := details["changes"].([]interface{}); ok {
			changes = make([]map[string]interface{}, len(rawChanges))
			for i, c := range rawChanges {
				changes[i], _ = c.(map[string]interface{})
			}
		}
	}

	if len(changes) == 0 {
		return fmt.Errorf("nenhuma alteração para reverter")
	}

	log(fmt.Sprintf("🔄 Revertendo tributação IBS/CBS de %d produtos...", len(changes)))

	produtosEmpresa := rm.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	count := 0

	for _, change := range changes {
		productIDHex, _ := change["productId"].(string)
		prevTribHex, _ := change["prevTribId"].(string)

		if productIDHex == "" {
			continue
		}

		productOID, err := primitive.ObjectIDFromHex(productIDHex)
		if err != nil {
			continue
		}

		var update bson.M
		if prevTribHex != "" {
			tribOID, _ := primitive.ObjectIDFromHex(prevTribHex)
			update = bson.M{"$set": bson.M{"TributacaoIbsCbsReferencia": tribOID}}
		} else {
			update = bson.M{"$unset": bson.M{"TributacaoIbsCbsReferencia": 1}}
		}

		result, err := produtosEmpresa.UpdateOne(ctx, bson.M{"_id": productOID}, update)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ Tributação IBS/CBS revertida para %d produtos", count))
	return nil
}

func (rm *RollbackManager) undoBulkActivate(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	products, ok := details["products"].([]map[string]interface{})
	if !ok {
		if rawProducts, ok := details["products"].([]interface{}); ok {
			products = make([]map[string]interface{}, len(rawProducts))
			for i, p := range rawProducts {
				products[i], _ = p.(map[string]interface{})
			}
		}
	}

	if len(products) == 0 {
		return fmt.Errorf("nenhum produto para reverter")
	}

	log(fmt.Sprintf("🔄 Revertendo status de %d produtos...", len(products)))

	produtosServicos := rm.conn.GetCollection(database.CollectionProdutosServicos)
	count := 0

	for _, prod := range products {
		idHex, _ := prod["id"].(string)
		wasActive, _ := prod["wasActive"].(bool)

		if idHex == "" {
			continue
		}

		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			continue
		}

		result, err := produtosServicos.UpdateOne(ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{"Ativo": wasActive}},
		)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ Status revertido para %d produtos", count))
	return nil
}

func (rm *RollbackManager) undoZeroStock(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	stocks, ok := details["stocks"].([]map[string]interface{})
	if !ok {
		if rawStocks, ok := details["stocks"].([]interface{}); ok {
			stocks = make([]map[string]interface{}, len(rawStocks))
			for i, s := range rawStocks {
				stocks[i], _ = s.(map[string]interface{})
			}
		}
	}

	if len(stocks) == 0 {
		return fmt.Errorf("nenhum estoque para reverter")
	}

	log(fmt.Sprintf("🔄 Restaurando %d estoques...", len(stocks)))

	estoques := rm.conn.GetCollection(database.CollectionEstoques)
	count := 0

	for _, stock := range stocks {
		idHex, _ := stock["id"].(string)
		prevQty, _ := stock["prevQuantity"].(float64)

		if idHex == "" {
			continue
		}

		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			continue
		}

		result, err := estoques.UpdateOne(ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{"Quantidades.0.Quantidade": prevQty}},
		)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ %d estoques restaurados", count))
	return nil
}

func (rm *RollbackManager) undoZeroAllPrices(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	products, ok := details["products"].([]map[string]interface{})
	if !ok {
		if rawProducts, ok := details["products"].([]interface{}); ok {
			products = make([]map[string]interface{}, len(rawProducts))
			for i, p := range rawProducts {
				products[i], _ = p.(map[string]interface{})
			}
		}
	}

	if len(products) == 0 {
		return fmt.Errorf("nenhum preço para reverter")
	}

	log(fmt.Sprintf("🔄 Restaurando preços de %d produtos...", len(products)))

	produtosEmpresa := rm.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	count := 0

	for _, prod := range products {
		idHex, _ := prod["id"].(string)
		prevCost, _ := prod["prevCost"].(float64)
		prevSale, _ := prod["prevSale"].(float64)

		if idHex == "" {
			continue
		}

		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			continue
		}

		result, err := produtosEmpresa.UpdateOne(ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{
				"PrecosCustos.0.Valor": prevCost,
				"PrecosVendas.0.Valor": prevSale,
			}},
		)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ Preços restaurados para %d produtos", count))
	return nil
}

func (rm *RollbackManager) undoInvoiceKeyChange(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	invoiceID, _ := details["invoiceId"].(string)
	prevKey, _ := details["prevKey"].(string)

	if invoiceID == "" || prevKey == "" {
		return fmt.Errorf("dados insuficientes para reverter chave")
	}

	log(fmt.Sprintf("🔄 Restaurando chave da nota %s...", invoiceID))

	coll := rm.conn.GetCollection("Movimentacoes")

	oid, err := primitive.ObjectIDFromHex(invoiceID)
	if err != nil {
		return fmt.Errorf("ID de nota inválido: %w", err)
	}

	result, err := coll.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{"ChaveAcesso": prevKey}},
	)

	if err != nil {
		return fmt.Errorf("erro ao restaurar chave: %w", err)
	}

	if result.ModifiedCount > 0 {
		log("✅ Chave restaurada com sucesso!")
	} else {
		log("⚠️ Nota não encontrada ou chave já estava correta")
	}

	return nil
}

func (rm *RollbackManager) undoInvoiceStatusChange(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	invoiceID, _ := details["invoiceId"].(string)
	prevSituacao, _ := details["prevSituacao"].(map[string]interface{})

	if invoiceID == "" {
		return fmt.Errorf("ID de nota não informado")
	}

	log(fmt.Sprintf("🔄 Restaurando situação da nota %s...", invoiceID))

	coll := rm.conn.GetCollection("Movimentacoes")

	oid, err := primitive.ObjectIDFromHex(invoiceID)
	if err != nil {
		return fmt.Errorf("ID de nota inválido: %w", err)
	}

	update := bson.M{
		"$set": bson.M{
			"Situacao":             prevSituacao,
			"SituacaoMovimentacao": prevSituacao,
		},
		"$pop": bson.M{
			"Historicos": 1,
		},
	}

	result, err := coll.UpdateOne(ctx, bson.M{"_id": oid}, update)

	if err != nil {
		return fmt.Errorf("erro ao restaurar situação: %w", err)
	}

	if result.ModifiedCount > 0 {
		log("✅ Situação restaurada com sucesso!")
	} else {
		log("⚠️ Nota não encontrada")
	}

	return nil
}

func (rm *RollbackManager) undoEnableMEI(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	emitentes, ok := details["emitentes"].([]map[string]interface{})
	if !ok {
		if rawEmitentes, ok := details["emitentes"].([]interface{}); ok {
			emitentes = make([]map[string]interface{}, len(rawEmitentes))
			for i, e := range rawEmitentes {
				emitentes[i], _ = e.(map[string]interface{})
			}
		}
	}

	if len(emitentes) == 0 {
		return fmt.Errorf("nenhum emitente para reverter")
	}

	log(fmt.Sprintf("🔄 Revertendo MEI de %d emitentes...", len(emitentes)))

	pessoas := rm.conn.GetCollection(database.CollectionPessoas)
	count := 0

	for _, emit := range emitentes {
		idHex, _ := emit["id"].(string)
		wasEnabled, _ := emit["wasEnabled"].(bool)

		if idHex == "" {
			continue
		}

		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			continue
		}

		result, err := pessoas.UpdateOne(ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{"MicroempreendedorIndividual.Habilitado": wasEnabled}},
		)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ MEI revertido para %d emitentes", count))
	return nil
}

func (rm *RollbackManager) undoAdjustInventory(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	stocks, ok := details["stocks"].([]map[string]interface{})
	if !ok {
		if rawStocks, ok := details["stocks"].([]interface{}); ok {
			stocks = make([]map[string]interface{}, len(rawStocks))
			for i, s := range rawStocks {
				stocks[i], _ = s.(map[string]interface{})
			}
		}
	}

	if len(stocks) == 0 {
		return fmt.Errorf("nenhum estoque para reverter")
	}

	log(fmt.Sprintf("🔄 Restaurando %d estoques do ajuste de inventário...", len(stocks)))

	estoques := rm.conn.GetCollection(database.CollectionEstoques)
	count := 0

	for _, stock := range stocks {
		idHex, _ := stock["estoqueId"].(string)
		prevQty, _ := stock["prevQuantity"].(float64)

		if idHex == "" {
			continue
		}

		oid, err := primitive.ObjectIDFromHex(idHex)
		if err != nil {
			continue
		}

		result, err := estoques.UpdateOne(ctx,
			bson.M{"_id": oid},
			bson.M{"$set": bson.M{"Quantidades.0.Quantidade": prevQty}},
		)
		if err == nil && result.ModifiedCount > 0 {
			count++
		}
	}

	log(fmt.Sprintf("✅ %d estoques restaurados do ajuste de inventário", count))
	return nil
}

func (rm *RollbackManager) ClearHistory() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.history = make([]OperationRecord, 0)
}

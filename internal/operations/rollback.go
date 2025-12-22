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

// OperationType defines the type of operation for rollback
type OperationType string

const (
	OpInactivateProducts OperationType = "InactivateProducts"
	OpChangeTributation  OperationType = "ChangeTributation"
	OpChangeTribFederal  OperationType = "ChangeTributationFederal"
)

// OperationRecord stores information needed for rollback
type OperationRecord struct {
	ID        string                 `json:"id"`
	Type      OperationType          `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Label     string                 `json:"label"`
	Details   map[string]interface{} `json:"details"`
	Undoable  bool                   `json:"undoable"`
}

// RollbackManager handles operation history and rollback
type RollbackManager struct {
	history []OperationRecord
	mu      sync.RWMutex
	conn    *database.Connection
	maxOps  int
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(conn *database.Connection) *RollbackManager {
	return &RollbackManager{
		history: make([]OperationRecord, 0),
		conn:    conn,
		maxOps:  5, // Keep last 5 operations
	}
}

// RecordOperation adds an operation to history
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

	// Keep only last N operations
	if len(rm.history) > rm.maxOps {
		rm.history = rm.history[len(rm.history)-rm.maxOps:]
	}

	return id
}

// GetUndoableOperations returns operations that can be undone
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

// UndoOperation reverses a specific operation
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	switch target.Type {
	case OpInactivateProducts:
		err := rm.undoInactivateProducts(ctx, target.Details, log)
		if err != nil {
			return err
		}

	case OpChangeTributation, OpChangeTribFederal:
		err := rm.undoTributationChange(ctx, target.Details, log)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("tipo de operação não suportado para rollback: %s", target.Type)
	}

	// Remove from history after successful undo
	rm.history = append(rm.history[:targetIdx], rm.history[targetIdx+1:]...)

	return nil
}

// undoInactivateProducts re-activates products that were inactivated
func (rm *RollbackManager) undoInactivateProducts(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	productIDs, ok := details["productIds"].([]string)
	if !ok || len(productIDs) == 0 {
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

// undoTributationChange restores previous tributation
func (rm *RollbackManager) undoTributationChange(ctx context.Context, details map[string]interface{}, log LogFunc) error {
	changes, ok := details["changes"].([]map[string]interface{})
	if !ok || len(changes) == 0 {
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

// ClearHistory removes all history
func (rm *RollbackManager) ClearHistory() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.history = make([]OperationRecord, 0)
}

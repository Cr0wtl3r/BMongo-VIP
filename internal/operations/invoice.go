package operations

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InvoiceType represents the type of fiscal document
type InvoiceType string

const (
	InvoiceTypeNFe   InvoiceType = "NFe"
	InvoiceTypeNFCe  InvoiceType = "NFCe"
	InvoiceTypeSAT   InvoiceType = "SAT"
	InvoiceTypeDAV   InvoiceType = "DAV"
	InvoiceTypeDAVOS InvoiceType = "DAV-OS"
	InvoiceTypeNFSe  InvoiceType = "NFSe"
)

// InvoiceStatus represents the status of a fiscal document
type InvoiceStatus string

const (
	StatusConcluido   InvoiceStatus = "Concluído"
	StatusCancelado   InvoiceStatus = "Cancelado"
	StatusDenegado    InvoiceStatus = "Denegado"
	StatusPendente    InvoiceStatus = "Pendente"
	StatusEmDigitacao InvoiceStatus = "Em Digitação"
)

// StatusCodeMap maps status descriptions to codes (based on actual DB analysis)
// IMPORTANT: Code 1 = Aguardando, Code 2 = Concluído, Code 4 = Cancelado
var StatusCodeMap = map[InvoiceStatus]int{
	StatusConcluido:   2, // Concluído = 2 (NOT 1!)
	StatusCancelado:   4, // Cancelado = 4
	StatusDenegado:    3, // Atrasado = 3 (no Denegado sample found)
	StatusPendente:    1, // Aguardando = 1
	StatusEmDigitacao: 1, // Aguardando = 1
}

// GetCollectionForInvoiceType returns the collection name for a given invoice type
func GetCollectionForInvoiceType(invoiceType InvoiceType) string {
	// Debug findings: All movements seem to be in "Movimentacoes"
	return "Movimentacoes"
	/*
		switch invoiceType {
		case InvoiceTypeNFe:
			return "NotasFiscaisEletronicas"
		case InvoiceTypeNFCe:
			return "NotasFiscaisConsumidor"
		case InvoiceTypeSAT:
			return "CupomFiscalSat"
		case InvoiceTypeDAV, InvoiceTypeDAVOS:
			return "DocumentosAuxiliaresVenda"
		case InvoiceTypeNFSe:
			return "NotasFiscaisServico"
		default:
			return "NotasFiscaisEletronicas"
		}
	*/
}

// ChangeInvoiceKey updates an invoice's key (chave de acesso)
func (m *Manager) ChangeInvoiceKey(invoiceType string, oldKey string, newKey string, log LogFunc) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if m.state.ShouldStop() {
		return 0, fmt.Errorf("operação cancelada")
	}

	collName := GetCollectionForInvoiceType(InvoiceType(invoiceType))
	log(fmt.Sprintf("🔍 Buscando na coleção: %s", collName))
	log(fmt.Sprintf("🔑 Chave antiga: %s", oldKey))
	log(fmt.Sprintf("🔑 Chave nova: %s", newKey))

	// Get collection
	coll := m.conn.GetCollection(collName)

	// Find by old key
	filter := bson.M{"ChaveAcesso": oldKey}

	// Update to new key
	update := bson.M{
		"$set": bson.M{
			"ChaveAcesso": newKey,
		},
	}

	result, err := coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("erro ao atualizar chave: %w", err)
	}

	modified := int(result.ModifiedCount)
	log(fmt.Sprintf("✅ %d documento(s) atualizado(s)", modified))
	return modified, nil
}

// ChangeInvoiceStatus updates the status of an invoice
func (m *Manager) ChangeInvoiceStatus(invoiceType string, serie string, numero string, newStatus string, log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if m.state.ShouldStop() {
		return fmt.Errorf("operação cancelada")
	}

	collName := GetCollectionForInvoiceType(InvoiceType(invoiceType))
	log(fmt.Sprintf("🔍 Buscando na coleção: %s", collName))
	log(fmt.Sprintf("📄 Série: %s, Número: %s", serie, numero))
	log(fmt.Sprintf("📊 Nova situação: %s", newStatus))

	coll := m.conn.GetCollection(collName)

	// Default series to "1" if empty and not DAV
	if (invoiceType != string(InvoiceTypeDAV) && invoiceType != string(InvoiceTypeDAVOS)) && serie == "" {
		serie = "1"
	}

	// Build filter - for DAV/DAV-OS, don't use serie
	var filter bson.M

	// Create flexible filter for Numero (can be int or string)
	var numeroFilter interface{} = numero
	if numInt, err := helperToInt(numero); err == nil {
		numeroFilter = bson.M{"$in": bson.A{numero, numInt}}
	} else {
		numeroFilter = numero
	}

	if invoiceType == string(InvoiceTypeDAV) || invoiceType == string(InvoiceTypeDAVOS) {
		// DAV uses Numero directly
		filter = bson.M{
			"Numero": numeroFilter,
		}
	} else {
		// Other types use Serie.Numero + Numero
		// Serie can also be int or string
		var serieFilter interface{} = serie
		if serieInt, err := helperToInt(serie); err == nil {
			serieFilter = bson.M{"$in": bson.A{serie, serieInt}}
		} else {
			serieFilter = serie
		}

		filter = bson.M{
			"Serie":  serieFilter, // Changed from Serie.Numero to Serie based on debug
			"Numero": numeroFilter,
		}
	}

	// Get status code
	statusCode := StatusCodeMap[InvoiceStatus(newStatus)]
	if statusCode == 0 {
		statusCode = 1 // Default to Concluído
	}

	// Build situacao type name (MongoDB uses status names without accents)
	situacaoTypeName := newStatus
	switch newStatus {
	case "Concluído":
		situacaoTypeName = "Concluido"
	case "Em Digitação":
		situacaoTypeName = "EmDigitacao"
		// Others like Cancelado, Pendente, Denegado are already without accents
	}

	// Build history entry
	historicoEntry := bson.M{
		"SituacaoMovimentacao": bson.M{
			"_t": bson.A{
				"SituacaoMovimentacao",
				situacaoTypeName,
			},
			"Codigo":    statusCode,
			"Descricao": newStatus,
		},
		"NomeUsuario": "DigiTools",
		"Observacao":  "Situação definida manualmente via DigiTools",
		"DataHora":    primitive.NewDateTimeFromTime(time.Now()),
	}

	// Update document - Update BOTH "Situacao" AND "SituacaoMovimentacao" for consistency
	// The parser reads SituacaoMovimentacao first, so we must update it too
	situacaoObj := bson.M{
		"_t": bson.A{
			"SituacaoMovimentacao",
			situacaoTypeName,
		},
		"Codigo":    statusCode,
		"Descricao": newStatus,
	}

	update := bson.M{
		"$set": bson.M{
			"Situacao":             situacaoObj,
			"SituacaoMovimentacao": situacaoObj, // Also update this field!
		},
		"$push": bson.M{
			"Historicos": historicoEntry,
		},
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("erro ao atualizar situação: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("documento não encontrado")
	}

	log("✅ Situação atualizada com sucesso!")
	return nil
}

// GetInvoiceTypes returns available invoice types
func GetInvoiceTypes() []string {
	return []string{
		string(InvoiceTypeNFe),
		string(InvoiceTypeNFCe),
		string(InvoiceTypeSAT),
		string(InvoiceTypeDAV),
		string(InvoiceTypeDAVOS),
		string(InvoiceTypeNFSe),
	}
}

// GetInvoiceStatuses returns available statuses
func GetInvoiceStatuses() []string {
	return []string{
		string(StatusConcluido),
		string(StatusCancelado),
		string(StatusDenegado),
		string(StatusPendente),
		string(StatusEmDigitacao),
	}
}

// InvoiceDetails represents a simplified view of an invoice for preview
type InvoiceDetails struct {
	ID       string  `json:"id"` // MongoDB ObjectID hex
	Numero   string  `json:"numero"`
	Serie    string  `json:"serie"`
	Chave    string  `json:"chave"`
	Data     string  `json:"data"`
	Cliente  string  `json:"cliente"`
	Valor    float64 `json:"valor"`
	Situacao string  `json:"situacao"`
}

// GetInvoiceByKey finds an invoice by its key and returns simplified details
func (m *Manager) GetInvoiceByKey(invoiceType string, key string, log LogFunc) (*InvoiceDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collName := GetCollectionForInvoiceType(InvoiceType(invoiceType))
	coll := m.conn.GetCollection(collName)

	var result bson.M
	err := coll.FindOne(ctx, bson.M{"ChaveAcesso": key}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("nota não encontrada: %v", err)
	}

	return m.parseInvoiceDetails(result), nil
}

// GetInvoiceByNumber finds an invoice by series and number
func (m *Manager) GetInvoiceByNumber(invoiceType string, serie string, number string, log LogFunc) (*InvoiceDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collName := GetCollectionForInvoiceType(InvoiceType(invoiceType))
	coll := m.conn.GetCollection(collName)

	// Default series to "1" if empty and not DAV
	if (invoiceType != string(InvoiceTypeDAV) && invoiceType != string(InvoiceTypeDAVOS)) && serie == "" {
		serie = "1"
	}

	var filter bson.M

	// Flexible filters
	var numeroFilter interface{} = number
	if numInt, err := helperToInt(number); err == nil {
		numeroFilter = bson.M{"$in": bson.A{number, numInt}}
	} else {
		numeroFilter = number
	}

	if invoiceType == string(InvoiceTypeDAV) || invoiceType == string(InvoiceTypeDAVOS) {
		filter = bson.M{"Numero": numeroFilter}
	} else {
		var serieFilter interface{} = serie
		if serieInt, err := helperToInt(serie); err == nil {
			serieFilter = bson.M{"$in": bson.A{serie, serieInt}}
		} else {
			serieFilter = serie
		}

		filter = bson.M{
			"Serie":  serieFilter,
			"Numero": numeroFilter,
		}
	}

	var result bson.M
	err := coll.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("nota não encontrada: %v", err)
	}

	return m.parseInvoiceDetails(result), nil
}

// parseInvoiceDetails extracts common fields from the flexible MongoDB document
func (m *Manager) parseInvoiceDetails(doc bson.M) *InvoiceDetails {
	details := &InvoiceDetails{}

	// ID
	if id, ok := doc["_id"].(primitive.ObjectID); ok {
		details.ID = id.Hex()
	}

	// Numero
	details.Numero = getStringSafe(doc, "Numero")

	// Serie
	if serieObj, ok := doc["Serie"].(primitive.M); ok {
		details.Serie = getStringSafe(serieObj, "Numero")
	} else {
		details.Serie = getStringSafe(doc, "Serie")
	}

	// Chave
	details.Chave = getStringSafe(doc, "ChaveAcesso")

	// Data
	if dt, ok := doc["DataHoraEmissao"].(primitive.DateTime); ok {
		details.Data = dt.Time().Format("02/01/2006 15:04:05")
	} else if dt, ok := doc["DataEmissao"].(primitive.DateTime); ok {
		details.Data = dt.Time().Format("02/01/2006")
	}

	// Cliente (Destinatario ou Pessoa)
	if dest, ok := doc["Destinatario"].(primitive.M); ok {
		details.Cliente = getStringSafe(dest, "Nome")
	} else if pessoa, ok := doc["Pessoa"].(primitive.M); ok {
		details.Cliente = getStringSafe(pessoa, "Nome")
	}

	// Valor - Calculate from ItensBase (most reliable method based on DB analysis)
	if itens, ok := doc["ItensBase"].(primitive.A); ok && len(itens) > 0 {
		var total float64
		for _, item := range itens {
			if itemMap, ok := item.(primitive.M); ok {
				preco := toFloatSafe(itemMap["PrecoUnitario"])
				qtd := toFloatSafe(itemMap["Quantidade"])
				desconto := toFloatSafe(itemMap["DescontoProporcional"])
				total += (preco * qtd) - desconto
			}
		}
		details.Valor = total
	} else if val, ok := doc["ValorTotal"].(float64); ok {
		details.Valor = val
	} else if val, ok := doc["TotalNota"].(float64); ok {
		details.Valor = val
	} else if val, ok := doc["Total"].(float64); ok {
		details.Valor = val
	} else {
		// Fallback: Try MovimentacoesReferenciadas[0].Total
		if movRefs, ok := doc["MovimentacoesReferenciadas"].(primitive.A); ok && len(movRefs) > 0 {
			if movRef, ok := movRefs[0].(primitive.M); ok {
				details.Valor = toFloatSafe(movRef["Total"])
			}
		}
	}

	// Situacao - Try object or string
	if sit, ok := doc["SituacaoMovimentacao"].(primitive.M); ok {
		details.Situacao = getStringSafe(sit, "Descricao")
	} else if sit, ok := doc["Situacao"].(primitive.M); ok {
		details.Situacao = getStringSafe(sit, "Descricao")
	} else if sit, ok := doc["SituacaoMovimentacao"].(string); ok {
		details.Situacao = sit
	} else if sit, ok := doc["Status"].(string); ok {
		details.Situacao = sit
	} else if sit, ok := doc["Situacao"].(string); ok {
		details.Situacao = sit
	}

	return details
}

// Helper to safely get string from bson.M (handles string and numeric types)
func getStringSafe(m bson.M, key string) string {
	val := m[key]
	if v, ok := val.(string); ok {
		return v
	}
	if v, ok := val.(int); ok {
		return fmt.Sprintf("%d", v)
	}
	if v, ok := val.(int32); ok {
		return fmt.Sprintf("%d", v)
	}
	if v, ok := val.(int64); ok {
		return fmt.Sprintf("%d", v)
	}
	if v, ok := val.(float64); ok {
		return fmt.Sprintf("%.0f", v)
	}
	return ""
}

// helperToInt converts string to int if possible
func helperToInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// toFloatSafe converts various numeric types to float64 safely
func toFloatSafe(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case int:
		return float64(t)
	}
	return 0
}

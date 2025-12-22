package operations

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)


type InvoiceType string

const (
	InvoiceTypeNFe   InvoiceType = "NFe"
	InvoiceTypeNFCe  InvoiceType = "NFCe"
	InvoiceTypeSAT   InvoiceType = "SAT"
	InvoiceTypeDAV   InvoiceType = "DAV"
	InvoiceTypeDAVOS InvoiceType = "DAV-OS"
	InvoiceTypeNFSe  InvoiceType = "NFSe"
)


type InvoiceStatus string

const (
	StatusConcluido   InvoiceStatus = "Concluído"
	StatusCancelado   InvoiceStatus = "Cancelado"
	StatusDenegado    InvoiceStatus = "Denegado"
	StatusPendente    InvoiceStatus = "Pendente"
	StatusEmDigitacao InvoiceStatus = "Em Digitação"
)


var StatusCodeMap = map[InvoiceStatus]int{
	StatusConcluido:   2,
	StatusCancelado:   4,
	StatusDenegado:    3,
	StatusPendente:    1,
	StatusEmDigitacao: 1,
}


func GetCollectionForInvoiceType(invoiceType InvoiceType) string {

	return "Movimentacoes"
	
}


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


	coll := m.conn.GetCollection(collName)


	filter := bson.M{"ChaveAcesso": oldKey}


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


	if (invoiceType != string(InvoiceTypeDAV) && invoiceType != string(InvoiceTypeDAVOS)) && serie == "" {
		serie = "1"
	}


	var filter bson.M


	var numeroFilter interface{} = numero
	if numInt, err := helperToInt(numero); err == nil {
		numeroFilter = bson.M{"$in": bson.A{numero, numInt}}
	} else {
		numeroFilter = numero
	}

	if invoiceType == string(InvoiceTypeDAV) || invoiceType == string(InvoiceTypeDAVOS) {

		filter = bson.M{
			"Numero": numeroFilter,
		}
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


	statusCode := StatusCodeMap[InvoiceStatus(newStatus)]
	if statusCode == 0 {
		statusCode = 1
	}


	situacaoTypeName := newStatus
	switch newStatus {
	case "Concluído":
		situacaoTypeName = "Concluido"
	case "Em Digitação":
		situacaoTypeName = "EmDigitacao"

	}


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
			"SituacaoMovimentacao": situacaoObj,
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


func GetInvoiceStatuses() []string {
	return []string{
		string(StatusConcluido),
		string(StatusCancelado),
		string(StatusDenegado),
		string(StatusPendente),
		string(StatusEmDigitacao),
	}
}


type InvoiceDetails struct {
	ID       string  `json:"id"`
	Numero   string  `json:"numero"`
	Serie    string  `json:"serie"`
	Chave    string  `json:"chave"`
	Data     string  `json:"data"`
	Cliente  string  `json:"cliente"`
	Valor    float64 `json:"valor"`
	Situacao string  `json:"situacao"`
}


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


func (m *Manager) GetInvoiceByNumber(invoiceType string, serie string, number string, log LogFunc) (*InvoiceDetails, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collName := GetCollectionForInvoiceType(InvoiceType(invoiceType))
	coll := m.conn.GetCollection(collName)


	if (invoiceType != string(InvoiceTypeDAV) && invoiceType != string(InvoiceTypeDAVOS)) && serie == "" {
		serie = "1"
	}

	var filter bson.M


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


func (m *Manager) parseInvoiceDetails(doc bson.M) *InvoiceDetails {
	details := &InvoiceDetails{}


	if id, ok := doc["_id"].(primitive.ObjectID); ok {
		details.ID = id.Hex()
	}


	details.Numero = getStringSafe(doc, "Numero")


	if serieObj, ok := doc["Serie"].(primitive.M); ok {
		details.Serie = getStringSafe(serieObj, "Numero")
	} else {
		details.Serie = getStringSafe(doc, "Serie")
	}


	details.Chave = getStringSafe(doc, "ChaveAcesso")


	if dt, ok := doc["DataHoraEmissao"].(primitive.DateTime); ok {
		details.Data = dt.Time().Format("02/01/2006 15:04:05")
	} else if dt, ok := doc["DataEmissao"].(primitive.DateTime); ok {
		details.Data = dt.Time().Format("02/01/2006")
	}


	if dest, ok := doc["Destinatario"].(primitive.M); ok {
		details.Cliente = getStringSafe(dest, "Nome")
	} else if pessoa, ok := doc["Pessoa"].(primitive.M); ok {
		details.Cliente = getStringSafe(pessoa, "Nome")
	}


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

		if movRefs, ok := doc["MovimentacoesReferenciadas"].(primitive.A); ok && len(movRefs) > 0 {
			if movRef, ok := movRefs[0].(primitive.M); ok {
				details.Valor = toFloatSafe(movRef["Total"])
			}
		}
	}


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


func helperToInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}


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

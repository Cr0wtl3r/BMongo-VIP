package operations

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"time"

	"BMongo-VIP/internal/database"

	"compress/gzip"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InvoiceSummary struct {
	ID              string    `json:"id"`
	Numero          int64     `json:"numero"`
	DataHoraEmissao time.Time `json:"dataEmissao"`
	TomadorNome     string    `json:"tomadorNome"`
	Total           float64   `json:"total"`
}

type InvoiceEmitter struct {
	Nome       string `json:"nome"`
	Fantasia   string `json:"fantasia"`
	CpfCnpj    string `json:"cpfCnpj"`
	Ie         string `json:"ie"`
	Endereco   string `json:"endereco"`
	Cidade     string `json:"cidade"`
	Estado     string `json:"estado"`
	Telefone   string `json:"telefone"`
	LogoBase64 string `json:"logoBase64"`
}

type InvoiceTomador struct {
	Nome     string `json:"nome"`
	CpfCnpj  string `json:"cpfCnpj"`
	Ie       string `json:"ie"`
	Telefone string `json:"telefone"`
	Endereco string `json:"endereco"`
	Cidade   string `json:"cidade"`
	Uf       string `json:"uf"`
}

type InvoiceItem struct {
	Codigo    string  `json:"codigo"`
	Descricao string  `json:"descricao"`
	Total     float64 `json:"total"`
}

type InvoiceData struct {
	ID                    string         `json:"id"`
	Numero                int64          `json:"numero"`
	DataHoraEmissao       time.Time      `json:"dataEmissao"`
	Emitente              InvoiceEmitter `json:"emitente"`
	Tomador               InvoiceTomador `json:"tomador"`
	Itens                 []InvoiceItem  `json:"itens"`
	TotalDescontoAplicado float64        `json:"totalDescontoAplicado"`
	TotalOutrasDespesas   float64        `json:"totalOutrasDespesas"`
	Total                 float64        `json:"total"`
	Observacao            string         `json:"observacao"`
	FormaPagamento        string         `json:"formaPagamento"`
}

// GetManualInvoices returns a list of manual invoices
func (m *Manager) GetManualInvoices(limit int) ([]InvoiceSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	movimentacoes := m.conn.GetCollection(database.CollectionMovimentacoes)

	// Debug: Log that we are starting
	fmt.Println("🔎 Buscando notas manuais...")

	// Filter by Discriminator (_t)
	// Supports cases where it might be a single string or an array of strings
	filter := bson.M{
		"$or": []bson.M{
			{"_t": "NotaFiscalManual"},
			{"_t": bson.M{"$in": []string{"NotaFiscalManual"}}},
		},
	}

	opts := options.Find().SetSort(bson.M{"DataHoraEmissao": -1}).SetLimit(int64(limit))

	cursor, err := movimentacoes.Find(ctx, filter, opts)
	if err != nil {
		fmt.Println("Erro query:", err)
		return nil, fmt.Errorf("erro ao buscar notas manuais: %v", err)
	}
	defer cursor.Close(ctx)

	var results []InvoiceSummary
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			fmt.Println("Erro decode:", err)
			continue
		}

		id, _ := doc["_id"].(primitive.ObjectID)
		numero := getInt64(doc, "Numero")
		dataEmissao := getDate(doc, "DataHoraEmissao")

		// Extract Tomador Name (Pessoa.Nome)
		tomador := ""
		if pessoa, ok := doc["Pessoa"].(bson.M); ok {
			tomador = getString(pessoa, "Nome")
		}

		// Calculate total considering discounts/expenses
		// Check both "Itens" and "ItensBase"
		total := 0.0

		itensRaw := doc["Itens"]
		if itensRaw == nil {
			itensRaw = doc["ItensBase"]
		}

		if itens, ok := itensRaw.(primitive.A); ok {
			for _, itemRaw := range itens {
				if item, ok := itemRaw.(bson.M); ok {
					qtde := getFloat64(item, "Quantidade")

					// ValorUnitario might be "PrecoUnitario" in some versions?
					valorUnit := getFloat64(item, "ValorUnitario")
					if valorUnit == 0 {
						valorUnit = getFloat64(item, "PrecoUnitario")
					}

					total += qtde * valorUnit
				}
			}
		}
		total -= getFloat64(doc, "TotalDescontoAplicado")
		total += getFloat64(doc, "TotalOutrasDespesas")

		results = append(results, InvoiceSummary{
			ID:              id.Hex(),
			Numero:          numero,
			DataHoraEmissao: dataEmissao,
			TomadorNome:     tomador,
			Total:           total,
		})
	}

	return results, nil
}

// GetInvoiceData returns full data for a specific invoice
func (m *Manager) GetInvoiceData(invoiceID string) (*InvoiceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(invoiceID)
	if err != nil {
		return nil, fmt.Errorf("ID inválido: %v", err)
	}

	movimentacoes := m.conn.GetCollection(database.CollectionMovimentacoes)
	var doc bson.M
	err = movimentacoes.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		return nil, fmt.Errorf("nota não encontrada: %v", err)
	}

	// 1. Tomador
	tomador := InvoiceTomador{}
	if pessoa, ok := doc["Pessoa"].(bson.M); ok {
		tomador.Nome = getString(pessoa, "Nome")
		tomador.CpfCnpj = getString(pessoa, "CpfCnpj") // Check if directly here or in a subfield
		if tomador.CpfCnpj == "" {
			// Try "Documento" inside Pessoa
			tomador.CpfCnpj = getString(pessoa, "Documento")
		}
		tomador.Ie = getString(pessoa, "Ie")

		// Address logic - check for EnderecoPrincipal first
		if endPrinc, ok := pessoa["EnderecoPrincipal"].(bson.M); ok {
			tomador.Endereco = fmt.Sprintf("%s, %s - %s",
				getString(endPrinc, "Logradouro"),
				getString(endPrinc, "Numero"),
				getString(endPrinc, "Bairro"))

			if municipio, ok := endPrinc["Municipio"].(bson.M); ok {
				tomador.Cidade = getString(municipio, "Nome")
				if uf, ok := municipio["Uf"].(bson.M); ok {
					tomador.Uf = getString(uf, "Sigla")
				}
			}
		} else {
			// Fallback to list
			if enderecos, ok := pessoa["Enderecos"].(primitive.A); ok && len(enderecos) > 0 {
				if end, ok := enderecos[0].(bson.M); ok {
					tomador.Endereco = fmt.Sprintf("%s, %s - %s",
						getString(end, "Logradouro"),
						getString(end, "Numero"),
						getString(end, "Bairro"))
					tomador.Cidade = getString(end, "Cidade")
					tomador.Uf = getString(end, "UF")
				}
			}
		}

		// Phone logic
		tomador.Telefone = getString(pessoa, "TelefonePrincipal")
		if tomador.Telefone == "" {
			if telefones, ok := pessoa["Telefones"].(primitive.A); ok && len(telefones) > 0 {
				if tel, ok := telefones[0].(bson.M); ok {
					tomador.Telefone = fmt.Sprintf("(%s) %s", getString(tel, "Ddd"), getString(tel, "Numero"))
				}
			}
		}
	}

	// 2. Emitente (Empresa)
	emitente := InvoiceEmitter{}
	var empresaID primitive.ObjectID
	if empresa, ok := doc["Empresa"].(bson.M); ok {
		// Sometimes Empresa is embedded completely, sometimes it's a ref.
		if id, ok := empresa["_id"].(primitive.ObjectID); ok {
			empresaID = id
		}
		// Fallback to EmpresaReferencia if _id is zero or missing
		if empresaID.IsZero() {
			if ref, ok := doc["EmpresaReferencia"].(primitive.ObjectID); ok {
				empresaID = ref
			}
		}

		emitente.Nome = getString(empresa, "Nome")
		emitente.Fantasia = getString(empresa, "NomeFantasia")
		if emitente.Fantasia == "" {
			emitente.Fantasia = getString(empresa, "Fantasia")
		}
		emitente.CpfCnpj = getString(empresa, "CpfCnpj")
		emitente.Ie = getString(empresa, "InscricaoEstadual")
		if emitente.Ie == "" {
			emitente.Ie = getString(empresa, "Ie")
		}

		if enderecos, ok := empresa["Enderecos"].(primitive.A); ok && len(enderecos) > 0 {
			if end, ok := enderecos[0].(bson.M); ok {
				emitente.Endereco = fmt.Sprintf("%s, %s - %s",
					getString(end, "Logradouro"),
					getString(end, "Numero"),
					getString(end, "Bairro"))
				emitente.Cidade = getString(end, "Cidade")
				emitente.Estado = getString(end, "UF")
			}
		}

		if telefones, ok := empresa["Telefones"].(primitive.A); ok && len(telefones) > 0 {
			if tel, ok := telefones[0].(bson.M); ok {
				emitente.Telefone = fmt.Sprintf("(%s) %s", getString(tel, "Ddd"), getString(tel, "Numero"))
			}
		}
	}

	// 3. Fetch Logo from Pessoas collection
	if !empresaID.IsZero() {
		pessoas := m.conn.GetCollection(database.CollectionPessoas)
		var pDoc bson.M
		opts := options.FindOne().SetProjection(bson.M{"Imagem": 1, "CpfCnpj": 1, "InscricaoEstadual": 1})
		if err := pessoas.FindOne(ctx, bson.M{"_id": empresaID}, opts).Decode(&pDoc); err == nil {
			if imgBinary, ok := pDoc["Imagem"].(primitive.Binary); ok {
				emitente.LogoBase64 = base64.StdEncoding.EncodeToString(m.ungzipIfNeeded(imgBinary.Data))
			}
			// Better CNPJ/IE sync from Pessoa document if missing in historico
			if emitente.CpfCnpj == "" {
				emitente.CpfCnpj = getString(pDoc, "CpfCnpj")
			}
			if emitente.Ie == "" {
				emitente.Ie = getString(pDoc, "InscricaoEstadual")
			}
		}
	} else {
		// Try fallback search by CNPJ if we have it but no ID
		if emitente.CpfCnpj != "" {
			pessoas := m.conn.GetCollection(database.CollectionPessoas)
			var pDoc bson.M
			if err := pessoas.FindOne(ctx, bson.M{"CpfCnpj": emitente.CpfCnpj}).Decode(&pDoc); err == nil {
				if imgBinary, ok := pDoc["Imagem"].(primitive.Binary); ok {
					emitente.LogoBase64 = base64.StdEncoding.EncodeToString(m.ungzipIfNeeded(imgBinary.Data))
				}
				if emitente.Ie == "" {
					emitente.Ie = getString(pDoc, "InscricaoEstadual")
				}
			}
		}
	}

	// 4. Itens
	var itens []InvoiceItem
	total := 0.0

	itensRaw := doc["Itens"]
	if itensRaw == nil {
		itensRaw = doc["ItensBase"]
	}

	if itensArr, ok := itensRaw.(primitive.A); ok {
		for _, itemRaw := range itensArr {
			if item, ok := itemRaw.(bson.M); ok {
				qtde := getFloat64(item, "Quantidade")

				valorUnit := getFloat64(item, "ValorUnitario")
				if valorUnit == 0 {
					valorUnit = getFloat64(item, "PrecoUnitario")
				}

				valorTotal := qtde * valorUnit
				total += valorTotal

				// Get Descricao from embedded Product
				descricao := ""
				codigo := ""

				// Product structure varies
				if prod, ok := item["Produto"].(bson.M); ok {
					descricao = getString(prod, "Descricao")
					codigo = getString(prod, "CodigoInterno")
				} else if prodServ, ok := item["ProdutoServico"].(bson.M); ok {
					// Fallback for newer schemas (like the user sample)
					descricao = getString(prodServ, "Descricao")
					codigo = getString(prodServ, "CodigoInterno")
				} else {
					// Fallback if description is in item itself
					descricao = getString(item, "Descricao")
				}

				// Apply item discount if exists
				desconto := getFloat64(item, "ValorDesconto")
				valorTotal -= desconto

				itens = append(itens, InvoiceItem{
					Codigo:    codigo,
					Descricao: descricao,
					Total:     valorTotal,
				})
			}
		}
	}

	// 5. Forma de Pagamento
	formaPagamento := "À Vista" // Default
	if pagRec, ok := doc["PagamentoRecebimento"].(bson.M); ok {
		if parcelas, ok := pagRec["Parcelas"].(primitive.A); ok && len(parcelas) > 0 {
			if p0, ok := parcelas[0].(bson.M); ok {
				if historico, ok := p0["Historico"].(primitive.A); ok && len(historico) > 0 {
					// Usually the last one or first one? Let's take the first one (0)
					if h0, ok := historico[0].(bson.M); ok {
						if espPag, ok := h0["EspeciePagamento"].(bson.M); ok {
							if desc, ok := espPag["Descricao"].(string); ok {
								formaPagamento = desc
							}
						}
					}
				}
			}
		}
	} else if itensPag, ok := doc["ItensPagamentos"].(primitive.A); ok && len(itensPag) > 0 {
		// Fallback for NFe/NFCe structure
		if ip0, ok := itensPag[0].(bson.M); ok {
			if espPag, ok := ip0["EspeciePagamento"].(bson.M); ok {
				if desc, ok := espPag["Descricao"].(string); ok {
					formaPagamento = desc
				}
			}
		}
	}

	result := &InvoiceData{
		ID:                    invoiceID,
		Numero:                getInt64(doc, "Numero"),
		DataHoraEmissao:       getDate(doc, "DataHoraEmissao"),
		Emitente:              emitente,
		Tomador:               tomador,
		Itens:                 itens,
		TotalDescontoAplicado: getFloat64(doc, "TotalDescontoAplicado"),
		TotalOutrasDespesas:   getFloat64(doc, "TotalOutrasDespesas"),
		Total:                 total - getFloat64(doc, "TotalDescontoAplicado") + getFloat64(doc, "TotalOutrasDespesas"),
		Observacao:            getString(doc, "Observacao"),
		FormaPagamento:        formaPagamento,
	}

	return result, nil
}

func getInt64(doc bson.M, key string) int64 {
	if val, ok := doc[key]; ok && val != nil {
		switch v := val.(type) {
		case int64:
			return v
		case int32:
			return int64(v)
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

func getFloat64(doc bson.M, key string) float64 {
	if val, ok := doc[key]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return v
		case int32:
			return float64(v)
		case int64:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 0.0
}

func getDate(doc bson.M, key string) time.Time {
	if val, ok := doc[key]; ok && val != nil {
		if t, ok := val.(primitive.DateTime); ok {
			return t.Time()
		}
		if t, ok := val.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}
func (m *Manager) ungzipIfNeeded(data []byte) []byte {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data
	}
	return m.decompressGzip(data)
}

func (m *Manager) decompressGzip(data []byte) []byte {
	b := bytes.NewReader(data)
	r, err := gzip.NewReader(b)
	if err != nil {
		return data
	}
	defer r.Close()

	res, err := ioutil.ReadAll(r)
	if err != nil {
		return data
	}
	return res
}

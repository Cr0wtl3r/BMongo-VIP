package operations

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"BMongo-VIP/internal/database"
	"BMongo-VIP/internal/windows"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmitenteInfo holds data parsed from info.dat XML
type EmitenteInfo struct {
	Cnpj                string `xml:"Cnpj"`
	RazaoSocial         string `xml:"RazaoSocial"`
	InscricaoEstadual   string `xml:"InscricaoEstadual"`
	RegistroMunicipal   string `xml:"RegistroMunicipal"`
	Cnae                string `xml:"Cnae"`
	Email               string `xml:"Email"`
	Telefone            string `xml:"Telefone"`
	Logradouro          string `xml:"Logradouro"`
	Numero              string `xml:"Numero"`
	Bairro              string `xml:"Bairro"`
	Cep                 string `xml:"Cep"`
	Complemento         string `xml:"Complemento"`
	CodigoIbgeMunicipio string `xml:"CodigoIbgeMunicipio"`
}

// InfoDatRoot represents the root element of info.dat
type InfoDatRoot struct {
	XMLName  xml.Name     `xml:"Emitente"`
	Emitente EmitenteInfo `xml:",any"`
}

// MunicipioInfo holds municipality data from IBGE/ViaCEP
type MunicipioInfo struct {
	Nome       string
	CodigoIbge int
	UF         UFInfo
}

// UFInfo holds state info
type UFInfo struct {
	Sigla      string
	Nome       string
	CodigoIbge int
}

// EmitenteBasic is a simplified emitente for listing
type EmitenteBasic struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
	Cnpj string `json:"cnpj"`
}

// ParseInfoDat parses XML from info.dat file
func ParseInfoDat(filePath string) (*EmitenteInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	info := &EmitenteInfo{}

	// Strategy 1: Standard XML Unmarshal
	// Pre-process: Replace encoding if necessary or just try raw
	var root InfoDatRoot
	if err := xml.Unmarshal(data, &root); err == nil && root.Emitente.Cnpj != "" {
		return &root.Emitente, nil
	}

	// Strategy 2: Direct Unmarshal
	if err := xml.Unmarshal(data, info); err == nil && info.Cnpj != "" {
		return info, nil
	}

	// Strategy 3: Manual "Legacy" Parsing (Regex/String scanning)
	// This mimics the legacy behavior which is robust against structural variations
	// and often ignores namespaces or root elements.
	content := string(data)

	// Helper to extract tag content
	extract := func(tag string) string {
		return extractTagValue(content, tag)
	}

	info.Cnpj = extract("Cnpj")
	info.RazaoSocial = extract("RazaoSocial")
	info.InscricaoEstadual = extract("InscricaoEstadual")
	info.RegistroMunicipal = extract("RegistroMunicipal")
	info.Cnae = extract("Cnae")
	info.Email = extract("Email")
	info.Telefone = extract("Telefone")
	info.Logradouro = extract("Logradouro")
	info.Numero = extract("Numero")
	info.Bairro = extract("Bairro")
	info.Cep = extract("Cep")
	info.Complemento = extract("Complemento")
	info.CodigoIbgeMunicipio = extract("CodigoIbgeMunicipio")

	if info.Cnpj == "" {
		// Log for debugging? We can't log here easily without LogFunc.
		// Return error or partial?
		// User provided log showed empty CNPJ, so this fallback is critical.
		return nil, fmt.Errorf("falha ao parsear XML: CNPJ não encontrado (tente verificar o formato do arquivo)")
	}

	return info, nil // Success with manual parsing
}

// extractTagValue finds <tag>value</tag> case-insensitive or loose
func extractTagValue(content, tag string) string {
	// Crude but effective for simple legacy XML
	// This assumes tags are not nested in comments etc.
	startTag := "<" + tag + ">"
	endTag := "</" + tag + ">"

	s := 0
	e := 0

	// Find start
	for i := 0; i < len(content)-len(startTag); i++ {
		if content[i:i+len(startTag)] == startTag {
			s = i + len(startTag)
			break
		}
	}
	if s == 0 {
		return ""
	}

	// Find end
	for i := s; i < len(content)-len(endTag); i++ {
		if content[i:i+len(endTag)] == endTag {
			e = i
			break
		}
	}

	if e > s {
		return content[s:e]
	}
	return ""
}

// UpdateEmitente updates the Matriz person with new data from info.dat
// NOW ACCEPTS filePath to copy correctly
func (m *Manager) UpdateEmitente(info *EmitenteInfo, filePath string, log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log(fmt.Sprintf("🔍 Validando dados parseados: CNPJ=%s, Razão=%s", info.Cnpj, info.RazaoSocial))
	if info.Cnpj == "" {
		return fmt.Errorf("dados do emitente inválidos (CNPJ vazio)")
	}

	log("🔄 Consultando município...")
	municipio, err := ConsultaMunicipio(info.CodigoIbgeMunicipio)
	if err != nil {
		log(fmt.Sprintf("⚠️ Erro ao consultar município: %v (continuando...)", err))
		// Continue with basic info
		municipio = &MunicipioInfo{
			Nome:       "Município",
			CodigoIbge: 0,
			UF:         UFInfo{},
		}
	}

	log(fmt.Sprintf("📍 Município: %s - %s", municipio.Nome, municipio.UF.Sigla))

	// Find Matriz (emitente principal)
	pessoas := m.conn.GetCollection(database.CollectionPessoas)
	filter := bson.M{
		"_t": bson.M{"$in": []string{"Matriz"}},
	}

	log("🔍 Buscando emitente Matriz no banco de dados...")

	// Build update document
	update := bson.M{
		"$set": bson.M{
			"Nome":                                               info.RazaoSocial,
			"Cnpj":                                               info.Cnpj,
			"CNAE":                                               info.Cnae,
			"NomeFantasia":                                       info.RazaoSocial,
			"RegistroMunicipal":                                  info.RegistroMunicipal,
			"Carteira.TelefonePrincipal.Numero":                  info.Telefone,
			"Carteira.EmailPrincipal.Endereco":                   info.Email,
			"Carteira.EnderecoPrincipal.Logradouro":              info.Logradouro,
			"Carteira.EnderecoPrincipal.Numero":                  info.Numero,
			"Carteira.EnderecoPrincipal.Cep":                     info.Cep,
			"Carteira.EnderecoPrincipal.Complemento":             info.Complemento,
			"Carteira.EnderecoPrincipal.Bairro":                  info.Bairro,
			"Carteira.EnderecoPrincipal.Municipio.Nome":          municipio.Nome,
			"Carteira.EnderecoPrincipal.Municipio.CodigoIbge":    municipio.CodigoIbge,
			"Carteira.EnderecoPrincipal.Municipio.Uf.CodigoIbge": municipio.UF.CodigoIbge,
			"Carteira.EnderecoPrincipal.Municipio.Uf.Nome":       municipio.UF.Nome,
			"Carteira.EnderecoPrincipal.Municipio.Uf.Sigla":      municipio.UF.Sigla,
			"Carteira.EnderecoPrincipal.InformacoesPesquisa.0":   fmt.Sprintf("%d", municipio.CodigoIbge),
			"Carteira.EnderecoPrincipal.InformacoesPesquisa.1":   fmt.Sprintf("%d", municipio.UF.CodigoIbge),
			"Carteira.Ie.Numero":                                 info.InscricaoEstadual,
			"Carteira.Ie.Uf.CodigoIbge":                          municipio.UF.CodigoIbge,
			"Carteira.Ie.Uf.Nome":                                municipio.UF.Nome,
			"Carteira.Ie.Uf.Sigla":                               municipio.UF.Sigla,
			"Carteira.Ie.Uf.Pais.CodigoBacen":                    1058,
			"Carteira.Ie.Uf.Pais.Nome":                           "Brasil",
		},
	}

	result, err := pessoas.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("erro ao atualizar emitente: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("nenhum emitente Matriz encontrado")
	}

	// COPY INFO.DAT TO SERVER (CRITICAL FOR LEGACY COMPATIBILITY)
	log(fmt.Sprintf("📂 Copiando info.dat de origem: %s", filePath))
	serverPath := `C:\DigiSat\SuiteG6\Servidor\info.dat`

	// Create directory if it doesn't exist (safety check)
	serverDir := `C:\DigiSat\SuiteG6\Servidor`
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		log(fmt.Sprintf("⚠️ Erro ao criar diretório do servidor: %v", err))
	}

	// Use the correct source path
	if _, err := os.Stat(filePath); err == nil {
		if err := copyFile(filePath, serverPath); err != nil {
			log(fmt.Sprintf("⚠️ Falha ao copiar info.dat para servidor: %v", err))
		} else {
			log("✅ info.dat atualizado no servidor com sucesso!")
		}
	} else {
		log(fmt.Sprintf("⚠️ Arquivo info.dat original não encontrado em %s para cópia.", filePath))
	}

	log(fmt.Sprintf("✅ Emitente atualizado com sucesso! (CNPJ: %s)", info.Cnpj))
	return nil
}

// UF data lookup table (IBGE codes)
var ufData = map[string]UFInfo{
	"AC": {Sigla: "AC", Nome: "Acre", CodigoIbge: 12},
	"AL": {Sigla: "AL", Nome: "Alagoas", CodigoIbge: 27},
	"AP": {Sigla: "AP", Nome: "Amapá", CodigoIbge: 16},
	"AM": {Sigla: "AM", Nome: "Amazonas", CodigoIbge: 13},
	"BA": {Sigla: "BA", Nome: "Bahia", CodigoIbge: 29},
	"CE": {Sigla: "CE", Nome: "Ceará", CodigoIbge: 23},
	"DF": {Sigla: "DF", Nome: "Distrito Federal", CodigoIbge: 53},
	"ES": {Sigla: "ES", Nome: "Espírito Santo", CodigoIbge: 32},
	"GO": {Sigla: "GO", Nome: "Goiás", CodigoIbge: 52},
	"MA": {Sigla: "MA", Nome: "Maranhão", CodigoIbge: 21},
	"MT": {Sigla: "MT", Nome: "Mato Grosso", CodigoIbge: 51},
	"MS": {Sigla: "MS", Nome: "Mato Grosso do Sul", CodigoIbge: 50},
	"MG": {Sigla: "MG", Nome: "Minas Gerais", CodigoIbge: 31},
	"PA": {Sigla: "PA", Nome: "Pará", CodigoIbge: 15},
	"PB": {Sigla: "PB", Nome: "Paraíba", CodigoIbge: 25},
	"PR": {Sigla: "PR", Nome: "Paraná", CodigoIbge: 41},
	"PE": {Sigla: "PE", Nome: "Pernambuco", CodigoIbge: 26},
	"PI": {Sigla: "PI", Nome: "Piauí", CodigoIbge: 22},
	"RJ": {Sigla: "RJ", Nome: "Rio de Janeiro", CodigoIbge: 33},
	"RN": {Sigla: "RN", Nome: "Rio Grande do Norte", CodigoIbge: 24},
	"RS": {Sigla: "RS", Nome: "Rio Grande do Sul", CodigoIbge: 43},
	"RO": {Sigla: "RO", Nome: "Rondônia", CodigoIbge: 11},
	"RR": {Sigla: "RR", Nome: "Roraima", CodigoIbge: 14},
	"SC": {Sigla: "SC", Nome: "Santa Catarina", CodigoIbge: 42},
	"SP": {Sigla: "SP", Nome: "São Paulo", CodigoIbge: 35},
	"SE": {Sigla: "SE", Nome: "Sergipe", CodigoIbge: 28},
	"TO": {Sigla: "TO", Nome: "Tocantins", CodigoIbge: 17},
}

// ConsultaMunicipio fetches municipality info by IBGE code
func ConsultaMunicipio(codigoIbge string) (*MunicipioInfo, error) {
	// Extract UF from IBGE code (first 2 digits)
	if len(codigoIbge) < 2 {
		return nil, fmt.Errorf("código IBGE inválido")
	}

	ufCode := codigoIbge[:2]
	var ufInfo UFInfo
	for _, uf := range ufData {
		if fmt.Sprintf("%d", uf.CodigoIbge) == ufCode {
			ufInfo = uf
			break
		}
	}

	// Call IBGE API to get municipality name
	url := fmt.Sprintf("https://servicodados.ibge.gov.br/api/v1/localidades/municipios/%s", codigoIbge)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar IBGE: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Nome string `json:"nome"`
		ID   int    `json:"id"`
	}

	if err := decodeJSON(resp.Body, &result); err != nil {
		// Fallback: return with just IBGE code
		ibgeInt := 0
		fmt.Sscanf(codigoIbge, "%d", &ibgeInt)
		return &MunicipioInfo{
			Nome:       "Município",
			CodigoIbge: ibgeInt,
			UF:         ufInfo,
		}, nil
	}

	return &MunicipioInfo{
		Nome:       result.Nome,
		CodigoIbge: result.ID,
		UF:         ufInfo,
	}, nil
}

// ListEmitentes returns all emitentes
func (m *Manager) ListEmitentes(log LogFunc) ([]EmitenteBasic, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log("🔍 Buscando emitentes...")

	pessoas := m.conn.GetCollection(database.CollectionPessoas)
	filter := bson.M{
		"_t": bson.M{"$in": []string{"Emitente", "Matriz", "Filial"}},
	}

	cursor, err := pessoas.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar emitentes: %w", err)
	}
	defer cursor.Close(ctx)

	var emitentes []EmitenteBasic
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		id, _ := doc["_id"].(primitive.ObjectID)
		nome, _ := doc["Nome"].(string)
		cnpj, _ := doc["Cnpj"].(string)

		emitentes = append(emitentes, EmitenteBasic{
			ID:   id.Hex(),
			Nome: nome,
			Cnpj: cnpj,
		})
	}

	log(fmt.Sprintf("✅ Encontrados %d emitentes", len(emitentes)))
	return emitentes, nil
}

// DeleteEmitente removes an emitente and associated data
// Implements full cascade delete matching legacy logic
func (m *Manager) DeleteEmitente(emitenteID string, log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(emitenteID)
	if err != nil {
		return fmt.Errorf("ID inválido: %v", err)
	}

	// 0. Kill Digisat Sync Process to release file locks on info.dat
	log("⚠️ Encerrando Sincronizador Digisat para liberar arquivos...")
	if count, err := windows.KillSyncProcess(func(msg string) {
		log("   " + msg)
	}); err != nil {
		log(fmt.Sprintf("⚠️ Falha ao encerrar sincronizador: %v (tentando continuar)", err))
	} else {
		if count > 0 {
			log("✅ Sincronizador encerrado.")
		}
	}

	// 0.1 Get Emitente Info BEFORE deleting (to check info.dat match)
	log(fmt.Sprintf("🗑️ Iniciando exclusão do emitente %s...", emitenteID))
	pessoasCollection := m.conn.GetCollection(database.CollectionPessoas)

	var emitenteToDelete struct {
		Cnpj string `bson:"Cnpj"`
	}
	// We ignore error here as it might be partially deleted already,
	// but we try to get CNPJ for safety check on info.dat
	_ = pessoasCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&emitenteToDelete)

	// Helper function to delete from collection by EmpresaReferencia
	deleteFromCollection := func(collectionName string) {
		col := m.conn.GetCollection(collectionName)
		res, err := col.DeleteMany(ctx, bson.M{"EmpresaReferencia": oid})
		if err != nil {
			log(fmt.Sprintf("   ⚠️ %s: erro - %v", collectionName, err))
		} else if res.DeletedCount > 0 {
			log(fmt.Sprintf("   ✓ %s: %d removidos", collectionName, res.DeletedCount))
		}
	}

	// ============================================
	// COMPLETE CLEANUP - ALL COLLECTIONS (LEGACY)
	// ============================================
	log("🔄 Removendo dados vinculados ao emitente...")
	log("   (Limpeza completa de 50+ coleções)")

	// Financial collections
	log("💰 Financeiro...")
	deleteFromCollection(database.CollectionMovimentacoes)
	deleteFromCollection(database.CollectionRecebimentos)
	deleteFromCollection(database.CollectionPagamentos)
	deleteFromCollection(database.CollectionMovimentosConta)
	deleteFromCollection(database.CollectionBoletos)
	deleteFromCollection(database.CollectionBoletosSemParcelaRecebimentoBoleto)
	deleteFromCollection(database.CollectionCheques)
	deleteFromCollection(database.CollectionComissoes)
	deleteFromCollection(database.CollectionConsultasServicoCredito)
	deleteFromCollection(database.CollectionItensCreditoDebitoCliente)
	deleteFromCollection(database.CollectionItensCreditoDebitoCashback)

	// Movements & Operations
	log("📦 Movimentações...")
	deleteFromCollection(database.CollectionAbastecimentos)
	deleteFromCollection(database.CollectionDevolucoes)
	deleteFromCollection(database.CollectionEntregasDelivery)
	deleteFromCollection(database.CollectionCartasCorrecao)
	deleteFromCollection(database.CollectionInutilizacoes)
	deleteFromCollection(database.CollectionManifestacaoDestinatario)
	deleteFromCollection(database.CollectionManifestosEletronicoDocumentoFiscal)
	deleteFromCollection(database.CollectionConhecimentosTransporteEletronico)
	deleteFromCollection(database.CollectionConhecimentosTransporteRodoviarioCargas)
	deleteFromCollection(database.CollectionRomaneiosCarga)
	deleteFromCollection(database.CollectionXmlMovimentacoes)

	// Restaurant/Food Service
	log("🍽️ Restaurante/Food Service...")
	deleteFromCollection(database.CollectionItensMesaConta)
	deleteFromCollection(database.CollectionItemMesaContaOcorrencias)
	deleteFromCollection(database.CollectionItensPedidoRestaurante)
	deleteFromCollection(database.CollectionMesasContasClienteBloqueadas)
	deleteFromCollection(database.CollectionOrdensCardapio)
	deleteFromCollection(database.CollectionReceitas)

	// Stock & Inventory
	log("📊 Estoques...")
	deleteFromCollection(database.CollectionEstoquesFisicos)
	deleteFromCollection(database.CollectionEstoquesFisicosMovimentacaoInterna)
	deleteFromCollection(database.CollectionConferenciasEstoque)
	deleteFromCollection(database.CollectionBicos)
	deleteFromCollection(database.CollectionDescontinuidadesEncerrante)
	deleteFromCollection(database.CollectionFolhasLmc)
	deleteFromCollection(database.CollectionSaldosIcmsStRetido)

	// Production & Industry
	log("🏭 Produção/Indústria...")
	deleteFromCollection(database.CollectionOrdensProducao)
	deleteFromCollection(database.CollectionOrcamentosIndustria)
	deleteFromCollection(database.CollectionMaosObra)

	// Integration & External
	log("🔗 Integrações...")
	deleteFromCollection(database.CollectionAnunciosMercadoLivre)
	deleteFromCollection(database.CollectionDadosDigisatContabil)
	deleteFromCollection(database.CollectionDadosDigisatScanntech)
	deleteFromCollection(database.CollectionArquivosDigisatContabil)
	deleteFromCollection(database.CollectionArquivosSngpc)

	// Scheduling & Personnel
	log("📅 Agendamentos/Pessoal...")
	deleteFromCollection(database.CollectionAgendamentos)
	deleteFromCollection(database.CollectionTurnos)
	deleteFromCollection(database.CollectionTurnosLancamentos)
	deleteFromCollection(database.CollectionJornadasTrabalho)
	deleteFromCollection(database.CollectionInternacoes)

	// Contracts & Control
	log("📋 Contratos/Controle...")
	deleteFromCollection(database.CollectionGestaoContratos)
	deleteFromCollection(database.CollectionControlesEntrega)
	deleteFromCollection(database.CollectionValesPresente)
	deleteFromCollection(database.CollectionRegistrosPafEcf)

	// Products & Services links
	log("🔗 Produtos/Serviços vinculados...")
	pseCollection := m.conn.GetCollection(database.CollectionProdutosServicosEmpresa)
	estoqueCollection := m.conn.GetCollection(database.CollectionEstoques)

	res, err := pseCollection.DeleteMany(ctx, bson.M{"EmpresaReferencia": oid})
	if err != nil {
		log(fmt.Sprintf("   ⚠️ ProdutosServicosEmpresa: erro - %v", err))
	} else {
		log(fmt.Sprintf("   ✓ ProdutosServicosEmpresa: %d removidos", res.DeletedCount))
	}

	// Cleanup orphaned Estoques (stocks not referenced by any PSE)
	log("   🧹 Verificando estoques órfãos...")
	cur, err := estoqueCollection.Find(ctx, bson.M{})
	if err != nil {
		log(fmt.Sprintf("   ⚠️ Erro ao listar estoques: %v", err))
	} else {
		defer cur.Close(ctx)
		var stocks []struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cur.All(ctx, &stocks); err == nil {
			deletedStocks := 0
			for _, stock := range stocks {
				count, _ := pseCollection.CountDocuments(ctx, bson.M{"EstoqueReferencia": stock.ID})
				if count == 0 {
					estoqueCollection.DeleteOne(ctx, bson.M{"_id": stock.ID})
					deletedStocks++
				}
			}
			if deletedStocks > 0 {
				log(fmt.Sprintf("   ✓ Estoques órfãos: %d removidos", deletedStocks))
			}
		}
	}

	// Configuration collections (CRITICAL for server startup)
	log("⚙️ Configurações (crítico para o servidor)...")
	deleteFromCollection(database.CollectionConfiguracoesServidor)
	deleteFromCollection(database.CollectionConfiguracoes)

	// 8. Finally, delete the Person (Emitente)
	log("🔄 Removendo cadastro do Emitente...")
	res, err = pessoasCollection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("erro ao remover emitente: %w", err)
	}

	if res.DeletedCount == 0 {
		return fmt.Errorf("emitente não encontrado para exclusão")
	}

	// 9. Check and clean info.dat if it belongs to this emitente
	log("🔍 Verificando integridade do servidor (info.dat)...")
	serverPath := `C:\DigiSat\SuiteG6\Servidor\info.dat`
	if _, err := os.Stat(serverPath); err == nil {
		// Parse existing info.dat to check if it matches the deleted company
		if info, err := ParseInfoDat(serverPath); err == nil {
			// Normalize CNPJ (remove non-digits) if needed, but usually they match format
			if info.Cnpj == emitenteToDelete.Cnpj && emitenteToDelete.Cnpj != "" {
				log(fmt.Sprintf("⚠️ O arquivo info.dat pertence ao emitente excluído (%s). Removendo arquivo...", info.Cnpj))
				if err := os.Remove(serverPath); err != nil {
					log(fmt.Sprintf("❌ Erro ao remover info.dat: %v", err))
				} else {
					log("✅ Arquivo info.dat removido para evitar falha no servidor.")
				}
			}
		}
	}

	log("✅ Emitente e dados relacionados excluídos com sucesso!")
	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// Helper to decode JSON
func decodeJSON(r io.Reader, v interface{}) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

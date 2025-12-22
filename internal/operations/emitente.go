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

	"digisat-tools/internal/database"

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

	// Try parsing with root element
	var root InfoDatRoot
	if err := xml.Unmarshal(data, &root); err == nil {
		return &root.Emitente, nil
	}

	// Try direct parsing (simpler structure)
	var info EmitenteInfo
	if err := xml.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("erro ao parsear XML: %w", err)
	}

	return &info, nil
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

// UpdateEmitente updates the Matriz person with new data from info.dat
func (m *Manager) UpdateEmitente(info *EmitenteInfo, log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

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

	log("🔍 Buscando emitente Matriz...")

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

	log(fmt.Sprintf("✅ Emitente atualizado! (CNPJ: %s)", info.Cnpj))
	return nil
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
// WARNING: This feature is currently DISABLED because it breaks the Digisat server
// The legacy system has many hidden references that we haven't fully mapped
func (m *Manager) DeleteEmitente(emitenteID string, log LogFunc) error {
	// DISABLED - This feature breaks the Digisat server
	// Need deeper investigation of the legacy system before re-enabling
	log("⚠️ FUNCIONALIDADE DESABILITADA")
	return fmt.Errorf("❌ Exclusão de emitente está temporariamente desabilitada. Esta operação quebra o servidor Digisat devido a referências internas que ainda não foram mapeadas. Restaure o backup se necessário")
}

// Helper to decode JSON
func decodeJSON(r io.Reader, v interface{}) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

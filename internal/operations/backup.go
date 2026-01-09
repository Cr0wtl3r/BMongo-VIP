package operations

import (
	"BMongo-VIP/internal/windows"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type BackupResult struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Timestamp string `json:"timestamp"`
}

func (m *Manager) BackupDatabase(outputDir string, log LogFunc) (*BackupResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if m.state.ShouldStop() {
		return nil, fmt.Errorf("operação cancelada")
	}

	log("🔄 Iniciando backup do banco de dados...")

	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	port := "12220"
	dbName := "DigisatServer"

	if host == "" || user == "" || pass == "" {
		return nil, fmt.Errorf("variáveis de ambiente DB_HOST, DB_USER, DB_PASS devem estar definidas")
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(outputDir, fmt.Sprintf("backup_%s", timestamp))

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório de backup: %w", err)
	}

	log(fmt.Sprintf("📁 Diretório de backup: %s", backupPath))

	args := []string{
		fmt.Sprintf("--host=%s:%s", host, port),
		fmt.Sprintf("--username=%s", user),
		fmt.Sprintf("--password=%s", pass),
		"--authenticationDatabase=admin",
		fmt.Sprintf("--db=%s", dbName),
		fmt.Sprintf("--out=%s", backupPath),
	}

	log("🚀 Executando mongodump...")

	mongodumpPath := findMongoTool("mongodump")
	if mongodumpPath == "" {
		return nil, fmt.Errorf("mongodump não encontrado. Verifique se MongoDB Tools está instalado")
	}

	log(fmt.Sprintf("📍 Usando: %s", mongodumpPath))

	cmd := exec.CommandContext(ctx, mongodumpPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log(fmt.Sprintf("❌ Erro no mongodump: %s", string(output)))
		return nil, fmt.Errorf("erro ao executar mongodump: %w - %s", err, string(output))
	}

	log(string(output))

	var totalSize int64
	err = filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		log(fmt.Sprintf("⚠️ Erro ao calcular tamanho: %v", err))
	}

	result := &BackupResult{
		Path:      backupPath,
		Size:      totalSize,
		Timestamp: timestamp,
	}

	log(fmt.Sprintf("✅ Backup concluído! Tamanho: %.2f MB", float64(totalSize)/1024/1024))

	return result, nil
}

func (m *Manager) RestoreDatabase(backupPath string, dropExisting bool, log LogFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if m.state.ShouldStop() {
		return fmt.Errorf("operação cancelada")
	}

	log("🔄 Iniciando restauração do banco de dados...")

	var tempDir string
	var cleanupTemp bool
	if strings.HasSuffix(strings.ToLower(backupPath), ".zip") {
		log(fmt.Sprintf("📦 Detectado arquivo ZIP: %s", filepath.Base(backupPath)))

		var err error
		tempDir, err = os.MkdirTemp("", "digisat_restore_")
		if err != nil {
			return fmt.Errorf("erro ao criar pasta temporária: %w", err)
		}
		cleanupTemp = true
		defer func() {
			if cleanupTemp {
				log(fmt.Sprintf("🧹 Limpando pasta temporária: %s", tempDir))
				os.RemoveAll(tempDir)
			}
		}()

		log(fmt.Sprintf("📂 Extraindo para: %s", tempDir))
		if err := extractZip(backupPath, tempDir); err != nil {
			return fmt.Errorf("erro ao extrair ZIP: %w", err)
		}
		log("✅ ZIP extraído com sucesso!")

		backupPath = tempDir
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("caminho de backup não encontrado: %s", backupPath)
	}

	_ = tempDir
	_ = cleanupTemp

	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	port := "12220"

	if host == "" || user == "" || pass == "" {
		return fmt.Errorf("variáveis de ambiente DB_HOST, DB_USER, DB_PASS devem estar definidas")
	}

	log(fmt.Sprintf("📁 Restaurando de: %s", backupPath))

	useGzip := false
	files, _ := os.ReadDir(backupPath)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".bson.gz") || strings.HasSuffix(f.Name(), ".metadata.json.gz") {
			useGzip = true
			log("📦 Detectado formato comprimido (--gzip)")
			break
		}
	}

	digisatSubfolder := filepath.Join(backupPath, "DigisatServer")
	if _, err := os.Stat(digisatSubfolder); err == nil {
		log(fmt.Sprintf("📂 Encontrada pasta DigisatServer em: %s", digisatSubfolder))

	} else {

		if filepath.Base(backupPath) == "DigisatServer" {
			parentPath := filepath.Dir(backupPath)
			log(fmt.Sprintf("📂 Detectado: selecionou pasta DigisatServer, usando pasta pai: %s", parentPath))
			backupPath = parentPath
		} else {

			hasDbFiles := false
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".bson") || strings.HasSuffix(f.Name(), ".bson.gz") {
					hasDbFiles = true
					break
				}
			}
			if hasDbFiles {
				log("📂 Detectado backup com estrutura plana (arquivos direto na pasta)")

			}
		}
	}

	args := []string{
		fmt.Sprintf("--host=%s:%s", host, port),
		fmt.Sprintf("--username=%s", user),
		fmt.Sprintf("--password=%s", pass),
		"--authenticationDatabase=admin",
		"--verbose",
	}

	if useGzip {
		args = append(args, "--gzip")
		log("🗜️ Usando descompressão gzip")
	}

	if dropExisting {
		args = append(args, "--drop")
		log("⚠️ Opção --drop ativada: coleções existentes serão DELETADAS e recriadas")
	}

	if _, err := os.Stat(digisatSubfolder); os.IsNotExist(err) && filepath.Base(backupPath) != "DigisatServer" {

		args = append(args, "--db=DigisatServer")
		log("📋 Especificando banco: DigisatServer")
	}

	args = append(args, backupPath)

	log("🚀 Executando mongorestore...")

	mongorestorePath := findMongoTool("mongorestore")
	if mongorestorePath == "" {
		return fmt.Errorf("mongorestore não encontrado. Verifique se MongoDB Tools está instalado")
	}

	log(fmt.Sprintf("📍 Usando: %s", mongorestorePath))

	cmd := exec.CommandContext(ctx, mongorestorePath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log(fmt.Sprintf("❌ Erro no mongorestore: %s", string(output)))
		return fmt.Errorf("erro ao executar mongorestore: %w - %s", err, string(output))
	}

	log(string(output))
	log("✅ Restauração concluída com sucesso! Reiniciando serviços do Digisat...")

	// Reiniciar serviços do Digisat
	if _, err := windows.StartDigisatServices(log); err != nil {
		log(fmt.Sprintf("⚠️ Aviso: Falha ao reiniciar serviços: %v", err))
	}

	return nil
}

func ListBackups(backupDir string) ([]BackupResult, error) {
	var backups []BackupResult

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler diretório: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) > 7 && entry.Name()[:7] == "backup_" {
			path := filepath.Join(backupDir, entry.Name())
			_, err := entry.Info()
			if err != nil {
				continue
			}

			var size int64
			filepath.Walk(path, func(p string, i os.FileInfo, e error) error {
				if e == nil && !i.IsDir() {
					size += i.Size()
				}
				return nil
			})

			backups = append(backups, BackupResult{
				Path:      path,
				Size:      size,
				Timestamp: entry.Name()[7:],
			})
		}
	}

	return backups, nil
}

func findMongoTool(toolName string) string {

	commonPaths := []string{

		`C:\DigiSat\SuiteG6\MongoDB\bin`,

		`C:\Digisat\MongoDB\bin`,
		`C:\Digisat\Server\MongoDB\bin`,
		`C:\Program Files\Digisat\MongoDB\bin`,

		`C:\Program Files\MongoDB\Server\7.0\bin`,
		`C:\Program Files\MongoDB\Server\6.0\bin`,
		`C:\Program Files\MongoDB\Server\5.0\bin`,
		`C:\Program Files\MongoDB\Server\4.4\bin`,
		`C:\Program Files\MongoDB\Tools\100\bin`,
		`C:\mongodb\bin`,

		`.`,
	}

	toolExe := toolName + ".exe"

	if path, err := exec.LookPath(toolExe); err == nil {
		return path
	}

	for _, basePath := range commonPaths {
		fullPath := filepath.Join(basePath, toolExe)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {

		fpath := filepath.Join(destDir, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("caminho inválido no ZIP: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

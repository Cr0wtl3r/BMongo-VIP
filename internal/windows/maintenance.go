package windows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var digisatPorts = []struct {
	Port     string
	Protocol string
	Name     string
}{
	{"12220", "TCP", "MongoDB Digisat"},
	{"8080", "TCP", "API Digisat"},
	{"8801", "TCP", "Servidor Digisat"},
	{"8802", "TCP", "Servidor Digisat 2"},
	{"8803", "TCP", "Mobile Digisat"},
	{"1433", "TCP", "SQL Server"},
}

var digisatPaths = []string{
	`C:\DigiSat`,
	`C:\DigiSat\SuiteG6`,
	`C:\DigiSat\SuiteG6\MongoDB`,
	`C:\DigiSat\SuiteG6\Server`,
}

func findMongod() string {
	commonPaths := []string{
		`C:\DigiSat\SuiteG6\MongoDB\bin`,
		`C:\Digisat\MongoDB\bin`,
		`C:\Digisat\Server\MongoDB\bin`,
		`C:\Program Files\MongoDB\Server\7.0\bin`,
		`C:\Program Files\MongoDB\Server\6.0\bin`,
		`C:\Program Files\MongoDB\Server\5.0\bin`,
		`C:\Program Files\MongoDB\Server\4.4\bin`,
		`C:\mongodb\bin`,
	}

	if path, err := exec.LookPath("mongod.exe"); err == nil {
		return path
	}

	for _, basePath := range commonPaths {
		fullPath := filepath.Join(basePath, "mongod.exe")
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

func findMongoDBPath() string {
	commonPaths := []string{
		`C:\DigiSat\SuiteG6\MongoDB\Dados`,
		`C:\Digisat\MongoDB\data`,
		`C:\Digisat\Server\MongoDB\data`,
		`C:\data\db`,
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func RepairMongoDB(log func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	log("🔧 Iniciando reparo do MongoDB (Offline)...")

	mongodPath := findMongod()
	if mongodPath == "" {
		return fmt.Errorf("mongod.exe não encontrado")
	}
	log(fmt.Sprintf("📍 Encontrado: %s", mongodPath))

	dbPath := findMongoDBPath()
	if dbPath == "" {
		return fmt.Errorf("diretório de dados do MongoDB não encontrado")
	}
	log(fmt.Sprintf("📂 Diretório de dados: %s", dbPath))

	log("🛑 Parando serviços MongoDB...")
	mongoServices := []string{"MongoDB", "MongoDBDigisat", "DigisatMongoDB"}
	for _, svc := range mongoServices {
		stopCmd := exec.CommandContext(ctx, "net", "stop", svc)
		stopCmd.Run()
	}

	log("💀 Encerrando processos mongod...")
	killCmd := exec.CommandContext(ctx, "taskkill", "/F", "/IM", "mongod.exe")
	killCmd.Run()

	time.Sleep(3 * time.Second)

	log("🔧 Executando mongod --repair...")
	repairArgs := []string{
		"--repair",
		fmt.Sprintf("--dbpath=%s", dbPath),
	}

	repairCmd := exec.CommandContext(ctx, mongodPath, repairArgs...)
	output, err := repairCmd.CombinedOutput()

	if err != nil {
		log(fmt.Sprintf("⚠️ Saída do reparo: %s", string(output)))
		return fmt.Errorf("erro no reparo: %w - %s", err, string(output))
	}

	log(fmt.Sprintf("📋 Saída: %s", string(output)))

	log("▶️ Reiniciando serviços MongoDB...")
	for _, svc := range mongoServices {
		startCmd := exec.CommandContext(ctx, "net", "start", svc)
		if out, err := startCmd.CombinedOutput(); err == nil {
			log(fmt.Sprintf("✅ Serviço %s iniciado", svc))
		} else {
			log(fmt.Sprintf("⚠️ Não foi possível iniciar %s: %s", svc, string(out)))
		}
	}

	log("✅ Reparo do MongoDB concluído!")
	return nil
}

func RepairMongoDBActive(mongoURL string, log func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log("🔧 Iniciando reparo do MongoDB (Ativo)...")

	mongoPath := ""
	commonPaths := []string{
		`C:\DigiSat\SuiteG6\MongoDB\bin`,
		`C:\Program Files\MongoDB\Server\7.0\bin`,
		`C:\Program Files\MongoDB\Server\6.0\bin`,
		`C:\Program Files\MongoDB\Server\5.0\bin`,
		`C:\Program Files\MongoDB\Server\4.4\bin`,
	}

	for _, basePath := range commonPaths {
		fullPath := filepath.Join(basePath, "mongo.exe")
		if _, err := os.Stat(fullPath); err == nil {
			mongoPath = fullPath
			break
		}
	}

	if mongoPath == "" {
		if path, err := exec.LookPath("mongo.exe"); err == nil {
			mongoPath = path
		}
	}

	if mongoPath == "" {
		mongoshPath := ""
		for _, basePath := range commonPaths {
			fullPath := filepath.Join(basePath, "mongosh.exe")
			if _, err := os.Stat(fullPath); err == nil {
				mongoshPath = fullPath
				break
			}
		}
		if mongoshPath == "" {
			if path, err := exec.LookPath("mongosh.exe"); err == nil {
				mongoshPath = path
			}
		}
		mongoPath = mongoshPath
	}

	if mongoPath == "" {
		return fmt.Errorf("mongo.exe ou mongosh.exe não encontrado")
	}

	log(fmt.Sprintf("📍 Usando: %s", mongoPath))

	repairScript := `db.runCommand({repairDatabase: 1})`

	args := []string{
		mongoURL,
		"--eval", repairScript,
	}

	cmd := exec.CommandContext(ctx, mongoPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "deprecated") || strings.Contains(string(output), "ok") {
			log(fmt.Sprintf("⚠️ Comando executado com avisos: %s", string(output)))
		} else {
			log(fmt.Sprintf("❌ Erro: %s", string(output)))
			return fmt.Errorf("erro no reparo: %w", err)
		}
	}

	log(fmt.Sprintf("📋 Resultado: %s", string(output)))
	log("✅ Reparo do MongoDB (Ativo) concluído!")

	return nil
}

func ReleaseFirewallPorts(log func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log("🔥 Liberando portas no Firewall do Windows...")

	for _, port := range digisatPorts {
		ruleName := fmt.Sprintf("Digisat_%s_%s", port.Name, port.Port)
		ruleName = strings.ReplaceAll(ruleName, " ", "_")

		log(fmt.Sprintf("📌 Adicionando regra: %s (Porta %s/%s)", port.Name, port.Port, port.Protocol))

		deleteCmd := exec.CommandContext(ctx, "netsh", "advfirewall", "firewall", "delete", "rule", fmt.Sprintf("name=%s", ruleName))
		deleteCmd.Run()

		inArgs := []string{
			"advfirewall", "firewall", "add", "rule",
			fmt.Sprintf("name=%s_IN", ruleName),
			"dir=in",
			"action=allow",
			fmt.Sprintf("protocol=%s", port.Protocol),
			fmt.Sprintf("localport=%s", port.Port),
		}

		inCmd := exec.CommandContext(ctx, "netsh", inArgs...)
		if out, err := inCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("⚠️ Erro na regra de entrada: %s", string(out)))
		} else {
			log(fmt.Sprintf("  ✅ Regra de entrada criada"))
		}

		outArgs := []string{
			"advfirewall", "firewall", "add", "rule",
			fmt.Sprintf("name=%s_OUT", ruleName),
			"dir=out",
			"action=allow",
			fmt.Sprintf("protocol=%s", port.Protocol),
			fmt.Sprintf("localport=%s", port.Port),
		}

		outCmd := exec.CommandContext(ctx, "netsh", outArgs...)
		if out, err := outCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("⚠️ Erro na regra de saída: %s", string(out)))
		} else {
			log(fmt.Sprintf("  ✅ Regra de saída criada"))
		}
	}

	log("✅ Portas liberadas no Firewall!")
	return nil
}

func AllowSecurityExclusions(log func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log("🛡️ Configurando exclusões de segurança...")

	log("📁 Adicionando exclusões no Windows Defender...")
	for _, path := range digisatPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log(fmt.Sprintf("  ⏭️ Pasta não existe, ignorando: %s", path))
			continue
		}

		psCmd := fmt.Sprintf("Add-MpPreference -ExclusionPath '%s'", path)
		cmd := exec.CommandContext(ctx, "powershell", "-Command", psCmd)

		if out, err := cmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("  ⚠️ Aviso ao adicionar exclusão %s: %s", path, string(out)))
		} else {
			log(fmt.Sprintf("  ✅ Exclusão adicionada: %s", path))
		}
	}

	exeExclusions := []string{
		`C:\DigiSat\SuiteG6\MongoDB\bin\mongod.exe`,
		`C:\DigiSat\SuiteG6\Server\ServidorDigisat.exe`,
		`C:\DigiSat\SuiteG6\Client\SistemaDigisat.exe`,
		`C:\DigiSat\SuiteG6\Sincronizador\DigisatSync.exe`,
	}

	log("📦 Adicionando exclusões de processos...")
	for _, exePath := range exeExclusions {
		if _, err := os.Stat(exePath); os.IsNotExist(err) {
			continue
		}

		psCmd := fmt.Sprintf("Add-MpPreference -ExclusionProcess '%s'", exePath)
		cmd := exec.CommandContext(ctx, "powershell", "-Command", psCmd)

		if out, err := cmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("  ⚠️ Aviso: %s", string(out)))
		} else {
			log(fmt.Sprintf("  ✅ Processo excluído: %s", filepath.Base(exePath)))
		}
	}

	log("🔓 Configurando permissões NTFS...")
	for _, path := range digisatPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		icaclsArgs := []string{path, "/grant", "Everyone:(OI)(CI)F", "/T", "/Q"}
		cmd := exec.CommandContext(ctx, "icacls", icaclsArgs...)

		if out, err := cmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("  ⚠️ Aviso permissão %s: %s", path, string(out)))
		} else {
			log(fmt.Sprintf("  ✅ Permissões configuradas: %s", path))
		}
	}

	log("✅ Configurações de segurança aplicadas!")
	return nil
}

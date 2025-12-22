package windows

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)


type DigiService struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}


type DigiProcess struct {
	Name string `json:"name"`
	PID  int    `json:"pid"`
}


var digisatServiceNames = []string{
	"DigiMonitor",
	"DigiServer",
	"DigiBackup",
	"DigisatService",
	"DigisatServer",
	"DigisatMobile",
	"DigisatSync",
}


var digisatProcessNames = []string{
	"DigiMonitor",
	"DigiServer",
	"DigiBackup",
	"Digisat",
	"DigisatPDV",
	"DigisatMobile",
	"DigisatRetaguarda",
	"DigisatSync",


}


func KillSyncProcess(log func(string)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log("🔄 Encerrando Sincronizador Digisat...")

	name := "DigisatSync"
	killed := 0


	checkCmd := exec.CommandContext(ctx, "tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s.exe", name))
	checkOutput, _ := checkCmd.CombinedOutput()

	if strings.Contains(string(checkOutput), name) {
		killCmd := exec.CommandContext(ctx, "taskkill", "/F", "/IM", fmt.Sprintf("%s.exe", name))
		if out, err := killCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("⚠️ Falha ao matar %s: %v (%s)", name, err, string(out)))
			return 0, err
		}
		log(fmt.Sprintf("✅ %s encerrado.", name))
		killed = 1
	} else {
		log(fmt.Sprintf("ℹ️ %s não estava rodando.", name))
	}


	log("🔄 Parando serviço DigisatSync...")
	exec.CommandContext(ctx, "net", "stop", "DigisatSync").Run()

	return killed, nil
}


func GetDigisatServices() ([]DigiService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var services []DigiService

	for _, name := range digisatServiceNames {

		cmd := exec.CommandContext(ctx, "sc", "query", name)
		output, err := cmd.CombinedOutput()

		if err != nil {

			continue
		}

		status := "UNKNOWN"
		outputStr := string(output)

		if strings.Contains(outputStr, "RUNNING") {
			status = "RUNNING"
		} else if strings.Contains(outputStr, "STOPPED") {
			status = "STOPPED"
		} else if strings.Contains(outputStr, "PAUSED") {
			status = "PAUSED"
		}

		services = append(services, DigiService{
			Name:        name,
			DisplayName: name,
			Status:      status,
		})
	}

	return services, nil
}


func StopDigisatServices(log func(string)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stoppedCount := 0

	log("🛑 Parando serviços Digisat...")

	for _, name := range digisatServiceNames {
		log(fmt.Sprintf("  Verificando serviço: %s", name))


		queryCmd := exec.CommandContext(ctx, "sc", "query", name)
		queryOutput, err := queryCmd.CombinedOutput()

		if err != nil {

			continue
		}

		if !strings.Contains(string(queryOutput), "RUNNING") {
			log(fmt.Sprintf("  ⏸️ %s já está parado", name))
			continue
		}


		log(fmt.Sprintf("  🔄 Parando %s...", name))
		stopCmd := exec.CommandContext(ctx, "net", "stop", name)
		stopOutput, err := stopCmd.CombinedOutput()

		if err != nil {
			log(fmt.Sprintf("  ⚠️ Erro ao parar %s: %s", name, string(stopOutput)))
			continue
		}

		log(fmt.Sprintf("  ✅ %s parado com sucesso", name))
		stoppedCount++
	}

	log(fmt.Sprintf("✅ %d serviço(s) parado(s)", stoppedCount))
	return stoppedCount, nil
}


func StartDigisatServices(log func(string)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	startedCount := 0

	log("▶️ Iniciando serviços Digisat...")

	for _, name := range digisatServiceNames {
		log(fmt.Sprintf("  Verificando serviço: %s", name))


		queryCmd := exec.CommandContext(ctx, "sc", "query", name)
		queryOutput, err := queryCmd.CombinedOutput()

		if err != nil {

			continue
		}

		if strings.Contains(string(queryOutput), "RUNNING") {
			log(fmt.Sprintf("  ▶️ %s já está rodando", name))
			continue
		}


		log(fmt.Sprintf("  🔄 Iniciando %s...", name))
		startCmd := exec.CommandContext(ctx, "net", "start", name)
		startOutput, err := startCmd.CombinedOutput()

		if err != nil {
			log(fmt.Sprintf("  ⚠️ Erro ao iniciar %s: %s", name, string(startOutput)))
			continue
		}

		log(fmt.Sprintf("  ✅ %s iniciado com sucesso", name))
		startedCount++
	}

	log(fmt.Sprintf("✅ %d serviço(s) iniciado(s)", startedCount))
	return startedCount, nil
}


func KillDigisatProcesses(log func(string)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	killedCount := 0

	log("💀 Encerrando processos Digisat...")

	for _, name := range digisatProcessNames {

		log(fmt.Sprintf("  Procurando processo: %s", name))


		checkCmd := exec.CommandContext(ctx, "tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s.exe", name))
		checkOutput, _ := checkCmd.CombinedOutput()

		if !strings.Contains(string(checkOutput), name) {
			continue
		}

		log(fmt.Sprintf("  🔄 Encerrando %s...", name))


		killCmd := exec.CommandContext(ctx, "taskkill", "/F", "/IM", fmt.Sprintf("%s.exe", name))
		killOutput, err := killCmd.CombinedOutput()

		if err != nil {
			log(fmt.Sprintf("  ⚠️ Erro ao encerrar %s: %s", name, string(killOutput)))
			continue
		}

		log(fmt.Sprintf("  ✅ %s encerrado", name))
		killedCount++
	}

	log(fmt.Sprintf("✅ %d processo(s) encerrado(s)", killedCount))
	return killedCount, nil
}


func GetDigisatProcesses() ([]DigiProcess, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var processes []DigiProcess


	cmd := exec.CommandContext(ctx, "tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("erro ao listar processos: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}


		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		processName := strings.Trim(parts[0], "\"")


		for _, dgName := range digisatProcessNames {
			if strings.Contains(strings.ToLower(processName), strings.ToLower(dgName)) {
				var pid int
				pidStr := strings.Trim(parts[1], "\"")
				fmt.Sscanf(pidStr, "%d", &pid)

				processes = append(processes, DigiProcess{
					Name: processName,
					PID:  pid,
				})
				break
			}
		}
	}

	return processes, nil
}

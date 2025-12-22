package windows

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DigiService represents a Digisat service
type DigiService struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}

// DigiProcess represents a Digisat process
type DigiProcess struct {
	Name string `json:"name"`
	PID  int    `json:"pid"`
}

// Common Digisat service names
var digisatServiceNames = []string{
	"DigiMonitor",
	"DigiServer",
	"DigiBackup",
	"DigisatService",
	"DigisatServer",
	"DigisatMobile",
	"DigisatSync",
}

// Common Digisat process names
var digisatProcessNames = []string{
	"DigiMonitor",
	"DigiServer",
	"DigiBackup",
	"Digisat",
	"DigisatPDV",
	"DigisatMobile",
	"DigisatRetaguarda",
	"DigisatSync",
	"MongoDB",
	"mongod",
}

// GetDigisatServices returns a list of Digisat services and their status
func GetDigisatServices() ([]DigiService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var services []DigiService

	for _, name := range digisatServiceNames {
		// Query service status using sc query
		cmd := exec.CommandContext(ctx, "sc", "query", name)
		output, err := cmd.CombinedOutput()

		if err != nil {
			// Service doesn't exist or error
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

// StopDigisatServices stops all Digisat services
func StopDigisatServices(log func(string)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stoppedCount := 0

	log("🛑 Parando serviços Digisat...")

	for _, name := range digisatServiceNames {
		log(fmt.Sprintf("  Verificando serviço: %s", name))

		// Check if service exists and is running
		queryCmd := exec.CommandContext(ctx, "sc", "query", name)
		queryOutput, err := queryCmd.CombinedOutput()

		if err != nil {
			// Service doesn't exist
			continue
		}

		if !strings.Contains(string(queryOutput), "RUNNING") {
			log(fmt.Sprintf("  ⏸️ %s já está parado", name))
			continue
		}

		// Stop the service
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

// StartDigisatServices starts all Digisat services
func StartDigisatServices(log func(string)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	startedCount := 0

	log("▶️ Iniciando serviços Digisat...")

	for _, name := range digisatServiceNames {
		log(fmt.Sprintf("  Verificando serviço: %s", name))

		// Check if service exists
		queryCmd := exec.CommandContext(ctx, "sc", "query", name)
		queryOutput, err := queryCmd.CombinedOutput()

		if err != nil {
			// Service doesn't exist
			continue
		}

		if strings.Contains(string(queryOutput), "RUNNING") {
			log(fmt.Sprintf("  ▶️ %s já está rodando", name))
			continue
		}

		// Start the service
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

// KillDigisatProcesses forcefully terminates all Digisat processes
func KillDigisatProcesses(log func(string)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	killedCount := 0

	log("💀 Encerrando processos Digisat...")

	for _, name := range digisatProcessNames {
		// Use taskkill to force kill the process
		log(fmt.Sprintf("  Procurando processo: %s", name))

		// First check if process is running using tasklist
		checkCmd := exec.CommandContext(ctx, "tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s.exe", name))
		checkOutput, _ := checkCmd.CombinedOutput()

		if !strings.Contains(string(checkOutput), name) {
			continue
		}

		log(fmt.Sprintf("  🔄 Encerrando %s...", name))

		// Kill the process
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

// GetDigisatProcesses returns a list of running Digisat processes
func GetDigisatProcesses() ([]DigiProcess, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var processes []DigiProcess

	// Get all running processes
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

		// Parse CSV format: "process.exe","PID","Session Name","Session#","Mem Usage"
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		processName := strings.Trim(parts[0], "\"")

		// Check if it's a Digisat process
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

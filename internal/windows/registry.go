package windows

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// RegistryManager handles Windows Registry operations
type RegistryManager struct{}

// NewRegistryManager creates a new registry manager
func NewRegistryManager() *RegistryManager {
	return &RegistryManager{}
}

// CleanDigisatRegistry removes specific Digisat registry keys
func (r *RegistryManager) CleanDigisatRegistry(log func(string)) error {
	log("🔄 Iniciando limpeza de registros do Windows...")

	// Keys to look for and delete
	// Based on typical Digisat removal tools (.reg files)

	// Example paths often used in these tools:
	// HKCU\Software\Digisat
	// HKCU\Software\Classes\VirtualStore\MACHINE\SOFTWARE\Wow6432Node\Digisat

	keysToCheck := []struct {
		root registry.Key
		path string
		name string
	}{
		{registry.CURRENT_USER, `Software\Digisat`, "HKCU\\Software\\Digisat"},
		{registry.CURRENT_USER, `Software\Classes\VirtualStore\MACHINE\SOFTWARE\Wow6432Node\Digisat`, "VirtualStore Digisat"},
		// Add more if found in legacy code analysis
	}

	count := 0
	for _, k := range keysToCheck {
		log(fmt.Sprintf("Verificando %s...", k.name))

		// Try to open key
		key, err := registry.OpenKey(k.root, k.path, registry.ALL_ACCESS)
		if err != nil {
			if strings.Contains(err.Error(), "The system cannot find the file specified") {
				continue // Key doesn't exist, which is fine
			}
			log(fmt.Sprintf("Erro ao acessar %s: %s", k.name, err.Error()))
			continue
		}
		key.Close() // Close before deleting

		// Delete key recursively
		err = r.deleteKeyRecursive(k.root, k.path)
		if err != nil {
			log(fmt.Sprintf("❌ Erro ao apagar %s: %s", k.name, err.Error()))
		} else {
			log(fmt.Sprintf("✅ %s removido com sucesso", k.name))
			count++
		}
	}

	log(fmt.Sprintf("Limpeza de registros concluída. %d chaves removidas.", count))
	return nil
}

// deleteKeyRecursive deletes a key and all its subkeys
func (r *RegistryManager) deleteKeyRecursive(root registry.Key, path string) error {
	// Open the key to list subkeys
	k, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		return err
	}
	defer k.Close()

	// Get all subkey names
	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return err
	}

	// Recursively delete subkeys
	for _, subkey := range subkeys {
		fullPath := path + "\\" + subkey
		if err := r.deleteKeyRecursive(root, fullPath); err != nil {
			return err
		}
	}

	// Close the key before deleting it
	k.Close()

	// Delete the key itself
	return registry.DeleteKey(root, path)
}

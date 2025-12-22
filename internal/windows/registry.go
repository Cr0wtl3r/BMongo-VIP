package windows

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)


type RegistryManager struct{}


func NewRegistryManager() *RegistryManager {
	return &RegistryManager{}
}


func (r *RegistryManager) CleanDigisatRegistry(log func(string)) error {
	log("🔄 Iniciando limpeza de registros do Windows...")








	keysToCheck := []struct {
		root registry.Key
		path string
		name string
	}{
		{registry.CURRENT_USER, `Software\Digisat`, "HKCU\\Software\\Digisat"},
		{registry.CURRENT_USER, `Software\Classes\VirtualStore\MACHINE\SOFTWARE\Wow6432Node\Digisat`, "VirtualStore Digisat"},

	}

	count := 0
	for _, k := range keysToCheck {
		log(fmt.Sprintf("Verificando %s...", k.name))


		key, err := registry.OpenKey(k.root, k.path, registry.ALL_ACCESS)
		if err != nil {
			if strings.Contains(err.Error(), "The system cannot find the file specified") {
				continue
			}
			log(fmt.Sprintf("Erro ao acessar %s: %s", k.name, err.Error()))
			continue
		}
		key.Close()


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


func (r *RegistryManager) deleteKeyRecursive(root registry.Key, path string) error {

	k, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		return err
	}
	defer k.Close()


	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return err
	}


	for _, subkey := range subkeys {
		fullPath := path + "\\" + subkey
		if err := r.deleteKeyRecursive(root, fullPath); err != nil {
			return err
		}
	}


	k.Close()


	return registry.DeleteKey(root, path)
}

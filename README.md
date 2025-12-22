# BMongo-VIP

Ferramenta de manutenção e administração para sistemas Digisat com MongoDB.

## 🚀 Funcionalidades

### Produtos

- Inativar produtos com estoque/preço zero
- Gerenciador avançado com filtros (NCM, descrição, estoque)
- Alterar tributação por NCM
- Zerar estoques e preços

### Notas Fiscais

- Alterar chave de acesso de documentos
- Alterar situação/status de notas
- Preview de notas antes de alteração

### Backup & Restore

- Backup do banco de dados (mongodump)
- Restaurar de pasta ou ZIP
- Suporte a backups comprimidos (.bson.gz)

### Emitentes

- Atualizar dados do emitente (info.dat)
- Consulta automática de município via IBGE
- Listagem de emitentes cadastrados

### Banco de Dados

- Limpeza de movimentações por data
- Limpeza completa (nova base)
- Buscar ObjectId no banco

### Windows

- Gerenciar serviços Digisat
- Encerrar processos
- Limpar registros do Windows

## 📦 Build

```bash
# Desenvolvimento
wails dev

# Build de produção
wails build -platform windows/amd64 -clean
```

O executável será gerado em `build/bin/BMongo-VIP.exe`

## ⚠️ Requisitos

- Windows 10/11
- MongoDB em execução
- Variáveis de ambiente:
  - `DB_HOST` - Host do MongoDB (ex: localhost:12220)
  - `DB_USER` - Usuário admin
  - `DB_PASS` - Senha

## 🔑 UAC

A aplicação requer privilégios de administrador para:

- Gerenciar serviços Windows
- Modificar registros
- Encerrar processos

## 📄 Licença

Uso exclusivo Digisat Sistemas.

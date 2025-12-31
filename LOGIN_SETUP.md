# 🔐 Como Configurar e Usar o Login no BMongo-VIP

## 📋 Resumo

O BMongo-VIP agora possui uma página de login segura que protege o acesso às funcionalidades. A senha é criptografada com **SHA-256** e não fica exposta no código compilado.

---

## 🚀 Primeiros Passos

### 1. Criar o Arquivo .env

Copie o arquivo de exemplo:

```powershell
Copy-Item .env.example .env
```

### 2. Gerar o Hash SHA-256 da Sua Senha

**Opção A - PowerShell (Recomendado):**

```powershell
$senha = "MinhaSenh@123"
$bytes = [System.Text.Encoding]::UTF8.GetBytes($senha)
$hash = [System.Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
$hashString = [System.BitConverter]::ToString($hash).Replace("-","").ToLower()
Write-Host "Hash SHA-256: $hashString"
```

**Opção B - Online:**

- Acesse: https://emn178.github.io/online-tools/sha256.html
- Digite sua senha
- Copie o hash gerado

**Opção C - Linux/Mac:**

```bash
echo -n "MinhaSenh@123" | shasum -a 256
```

### 3. Editar o Arquivo .env

Abra o arquivo `.env` e adicione o hash:

```
PASSWORD=seu_hash_sha256_aqui
```

Exemplo (para a senha "teste123"):

```
PASSWORD=ecd71870d1963316a97e3ac3408c9835ad8cf0f3c1bc703527c30265534f75ae
```

---

## 🏗️ Compilando a Aplicação

### ⚠️ IMPORTANTE

**NÃO use mais `wails build` diretamente!**

Use o script PowerShell que injeta a senha:

```powershell
.\build.ps1
```

O script irá:

1. Verificar se o `.env` existe
2. Extrair o hash da senha
3. Injetar o hash no executável durante a compilação
4. Gerar o executável em `.\build\bin\BMongo-VIP.exe`

---

## 🔒 Segurança

### ✅ O que está protegido:

- Hash SHA-256 é one-way (não pode ser revertido)
- Senha nunca fica em texto plano no código
- Arquivo `.env` não é versionado no Git
- Hash é injetado durante build via `ldflags`

### ⚠️ Recomendações:

- Use senhas fortes (12+ caracteres, maiúsculas, minúsculas, números, símbolos)
- Nunca compartilhe seu arquivo `.env`
- Mantenha o `.env` apenas no seu ambiente local

---

## 🧪 Testando

1. **Compile a aplicação:**

   ```powershell
   .\build.ps1
   ```

2. **Execute o programa:**

   ```powershell
   .\build\bin\BMongo-VIP.exe
   ```

3. **Teste o login:**
   - Digite a senha correta → Deve acessar a aplicação
   - Digite senha incorreta → Deve mostrar "Senha incorreta!"

---

## 🐛 Resolução de Problemas

### Erro: "PASSWORD não definido"

- Verifique se o arquivo `.env` existe
- Confirme que a linha `PASSWORD=...` está presente
- Verifique se não há espaços extras

### Login sempre falha

- Verifique se o hash no `.env` está correto
- Gere o hash novamente com o comando PowerShell
- Certifique-se de que não há espaços ou quebras de linha no hash

### Mensagem "Erro inesperado"

- Verifique os logs do console
- Recompile a aplicação com `.\build.ps1`

---

## 📝 Exemplo Completo

```powershell
# 1. Copiar .env.example
Copy-Item .env.example .env

# 2. Gerar hash da senha "admin123"
$senha = "admin123"
$bytes = [System.Text.Encoding]::UTF8.GetBytes($senha)
$hash = [System.Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
$hashString = [System.BitConverter]::ToString($hash).Replace("-","").ToLower()
Write-Host "PASSWORD=$hashString"

# 3. Copiar o output e adicionar ao .env manualmente
# Exemplo de output: PASSWORD=240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9

# 4. Compilar
.\build.ps1

# 5. Executar
.\build\bin\BMongo-VIP.exe
```

---

## 📚 Documentação Técnica

- **Algoritmo:** SHA-256
- **Formato do hash:** Hexadecimal lowercase (64 caracteres)
- **Injeção:** Via `-ldflags` durante build do Wails
- **Fonte:** Variável `PASSWORD` no `.env` ou compilada no binário

---

## 🔄 Alterando a Senha

1. Gere um novo hash SHA-256 para a nova senha
2. Atualize o arquivo `.env` com o novo hash
3. Recompile a aplicação com `.\build.ps1`
4. Use a nova senha ao fazer login

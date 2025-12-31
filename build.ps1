$envFile = "./.env"

if (-not (Test-Path $envFile)) {
    Write-Error "Erro: O arquivo .env não foi encontrado em $envFile"
    Write-Error "Certifique-se de que ele contém a variável PASSWORD."
    Write-Host ""
    Write-Host "Para criar o .env, copie .env.example e adicione seu hash SHA-256:"
    Write-Host "1. Copie: Copy-Item .env.example .env"
    Write-Host "2. Gere o hash: `$senha = 'SuaSenha'; [System.BitConverter]::ToString([System.Security.Cryptography.SHA256]::Create().ComputeHash([System.Text.Encoding]::UTF8.GetBytes(`$senha))).Replace('-','').ToLower()"
    Write-Host "3. Edite .env e adicione: PASSWORD=seu_hash_aqui"
    exit 1
}

$envContent = Get-Content $envFile -Raw

$passwordHash = ($envContent | Select-String -Pattern "PASSWORD=(.*)").Matches.Groups[1].Value.Trim()

if ([string]::IsNullOrEmpty($passwordHash)) {
    Write-Error "Erro: PASSWORD (hash) não encontrada no arquivo .env ou a linha está vazia."
    Write-Host ""
    Write-Host "Gere o hash SHA-256 da sua senha:"
    Write-Host "`$senha = 'SuaSenha'"
    Write-Host "`$bytes = [System.Text.Encoding]::UTF8.GetBytes(`$senha)"
    Write-Host "`$hash = [System.Security.Cryptography.SHA256]::Create().ComputeHash(`$bytes)"
    Write-Host "[System.BitConverter]::ToString(`$hash).Replace('-','').ToLower()"
    exit 1
}

Write-Host "✅ Hash da senha encontrado no .env" -ForegroundColor Green
Write-Host "Iniciando build do Wails..." -ForegroundColor Cyan

$envLines = Get-Content $envFile
$envMapEntries = ""
foreach ($line in $envLines) {
    $line = $line.Trim()
    if ($line -ne "" -and -not $line.StartsWith("#")) {
        $parts = $line.Split("=", 2)
        if ($parts.Length -eq 2) {
            $key = $parts[0].Trim()
            $val = $parts[1].Trim()
            
            if ($val.StartsWith('"') -and $val.EndsWith('"')) {
                $val = $val.Substring(1, $val.Length - 2)
            }
            
            $val = $val -replace '"', '\"'
            $envMapEntries += "`"$key`": `"$val`",`n`t`t"
        }
    }
}

$secretsFile = "secrets_gen.go"
$secretsContent = @"
package main

func init() {
	compiledPasswordHash = "$passwordHash"
	
	compiledEnv = map[string]string{
		$envMapEntries
	}
}
"@
Set-Content -Path $secretsFile -Value $secretsContent -Encoding UTF8

try {
    & wails build
}
finally {
    if (Test-Path $secretsFile) {
        Remove-Item $secretsFile
    }
}

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ Build concluído com sucesso!" -ForegroundColor Green
    Write-Host "📦 Executável gerado em: .\build\bin\BMongo-VIP.exe" -ForegroundColor Yellow
}
else {
    Write-Error "❌ Erro durante o build do Wails. Código de saída: $LASTEXITCODE"
}

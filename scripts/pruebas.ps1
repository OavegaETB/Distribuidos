# Prueba la API de la calculadora desde PowerShell (Windows).
# Primero inicie el servidor en otra terminal con: go run main.go
# Ejecutar con: powershell -ExecutionPolicy Bypass -File scripts\pruebas.ps1
$Url = "http://localhost:8080"

function Probar($titulo, $n1, $op, $n2) {
    Write-Host "-> $titulo" -ForegroundColor Cyan
    $cuerpo = @{ numero1 = $n1; numero2 = $n2; operador = $op } | ConvertTo-Json
    try {
        $r = Invoke-RestMethod -Method Post -Uri "$Url/calcular" -ContentType "application/json" -Body $cuerpo
        Write-Host ("   {0} {1} {2} = {3}" -f $r.numero1, $r.operador, $r.numero2, $r.resultado) -ForegroundColor Green
    } catch {
        $detalle = $_.ErrorDetails.Message
        Write-Host "   Error: $detalle" -ForegroundColor Red
    }
}

Probar "Suma 15 + 7"          15 "+" 7
Probar "Resta 20 - 8"         20 "-" 8
Probar "Multiplicacion 6 * 7" 6  "*" 7
Probar "Division 8 / 2"       8  "/" 2
Probar "Division por cero"    8  "/" 0
Probar "Operador invalido"    1  "%" 1
Probar "Fuera de rango"       1e308 "*" 10

Write-Host "-> Falta un campo" -ForegroundColor Cyan
try {
    Invoke-RestMethod -Method Post -Uri "$Url/calcular" -ContentType "application/json" -Body '{"numero1": 8, "operador": "/"}'
} catch {
    Write-Host "   Error: $($_.ErrorDetails.Message)" -ForegroundColor Red
}

Write-Host "-> Historial" -ForegroundColor Cyan
Invoke-RestMethod -Uri "$Url/historial" | Format-Table numero1, operador, numero2, resultado, fecha

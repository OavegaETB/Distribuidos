#!/usr/bin/env bash
# Prueba la API de la calculadora con curl (Linux, macOS o Git Bash en Windows).
# Primero inicie el servidor en otra terminal con: go run main.go
URL="${URL:-http://localhost:8080}"

probar() {
  echo "➜ $1"
  curl -s -X POST "$URL/calcular" -H "Content-Type: application/json" -d "$2"
  echo; echo
}

probar "Suma 15 + 7"           '{"numero1": 15, "numero2": 7, "operador": "+"}'
probar "Resta 20 - 8"          '{"numero1": 20, "numero2": 8, "operador": "-"}'
probar "Multiplicación 6 * 7"  '{"numero1": 6, "numero2": 7, "operador": "*"}'
probar "División 8 / 2"        '{"numero1": 8, "numero2": 2, "operador": "/"}'
probar "División por cero"     '{"numero1": 8, "numero2": 0, "operador": "/"}'
probar "Operador inválido"     '{"numero1": 1, "numero2": 1, "operador": "%"}'
probar "Fuera de rango"        '{"numero1": 1e308, "numero2": 10, "operador": "*"}'
probar "Falta un campo"        '{"numero1": 8, "operador": "/"}'
probar "Campo desconocido"     '{"numero1": 1, "numero2": 2, "operador": "+", "extra": true}'
probar "Tipo incorrecto"       '{"numero1": "ocho", "numero2": 2, "operador": "+"}'

echo "➜ Método GET en /calcular (405)"
curl -s "$URL/calcular"; echo; echo

echo "➜ Historial"
curl -s "$URL/historial"; echo

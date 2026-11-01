// Cliente de consola para la calculadora. Ejemplo: go run ./cliente 8 / 2
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// lo que se envía al servidor
type Solicitud struct {
	Numero1  float64 `json:"numero1"`
	Numero2  float64 `json:"numero2"`
	Operador string  `json:"operador"`
}

// lo que responde el servidor
type Operacion struct {
	Numero1   float64 `json:"numero1"`
	Numero2   float64 `json:"numero2"`
	Operador  string  `json:"operador"`
	Resultado float64 `json:"resultado"`
}

func main() {
	servidor := flag.String("servidor", "http://localhost:8080", "dirección del servidor de la calculadora")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Uso: go run ./cliente [-servidor URL] <numero1> <operador> <numero2>")
		fmt.Fprintln(os.Stderr, "Operadores: +  -  \"*\"  /")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 3 {
		flag.Usage()
		os.Exit(2)
	}

	n1, err1 := strconv.ParseFloat(flag.Arg(0), 64)
	n2, err2 := strconv.ParseFloat(flag.Arg(2), 64)
	if err1 != nil || err2 != nil {
		fmt.Fprintln(os.Stderr, "Error: numero1 y numero2 deben ser números")
		os.Exit(2)
	}

	op := Solicitud{Numero1: n1, Operador: flag.Arg(1), Numero2: n2}
	cuerpo, _ := json.Marshal(op)

	// enviar la operación al servidor
	clienteHTTP := &http.Client{Timeout: 5 * time.Second}
	resp, err := clienteHTTP.Post(*servidor+"/calcular", "application/json", bytes.NewReader(cuerpo))
	if err != nil {
		fmt.Fprintln(os.Stderr, "No se pudo conectar con el servidor:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&e)
		fmt.Fprintf(os.Stderr, "Error del servidor (%d): %s\n", resp.StatusCode, e.Error)
		os.Exit(1)
	}

	var res Operacion
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		fmt.Fprintln(os.Stderr, "Respuesta inválida:", err)
		os.Exit(1)
	}
	fmt.Printf("%g %s %g = %g\n", res.Numero1, res.Operador, res.Numero2, res.Resultado)
}

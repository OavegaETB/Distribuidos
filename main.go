// Calculadora distribuida: API REST en Go que resuelve operaciones por HTTP.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Operacion guarda los datos de una operación
type Operacion struct {
	Numero1   float64 `json:"numero1"`
	Numero2   float64 `json:"numero2"`
	Operador  string  `json:"operador"`
	Resultado float64 `json:"resultado"`
}

var (
	ErrDivisionPorCero  = errors.New("no se puede dividir entre cero")
	ErrOperadorInvalido = errors.New("operador no válido, use +, -, * o /")
	ErrFueraDeRango     = errors.New("el resultado es demasiado grande para representarlo")
)

// Calcular resuelve la operación según el operador
func (o *Operacion) Calcular() (float64, error) {
	switch o.Operador {
	case "+":
		o.Resultado = o.Numero1 + o.Numero2
	case "-":
		o.Resultado = o.Numero1 - o.Numero2
	case "*":
		o.Resultado = o.Numero1 * o.Numero2
	case "/":
		if o.Numero2 == 0 {
			return 0, ErrDivisionPorCero
		}
		o.Resultado = o.Numero1 / o.Numero2
	default:
		return 0, ErrOperadorInvalido
	}
	// si el número es demasiado grande da infinito
	if math.IsInf(o.Resultado, 0) || math.IsNaN(o.Resultado) {
		o.Resultado = 0
		return 0, ErrFueraDeRango
	}
	return o.Resultado, nil
}

type RegistroHistorial struct {
	Operacion
	Fecha time.Time `json:"fecha"`
}

// Historial de operaciones, compartido por todas las solicitudes.
// Se usa un mutex para que no se escriba al mismo tiempo.
type Historial struct {
	mu        sync.Mutex
	registros []RegistroHistorial
}

func (h *Historial) Agregar(op Operacion) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.registros = append(h.registros, RegistroHistorial{Operacion: op, Fecha: time.Now()})
}

// Listar devuelve una copia del historial
func (h *Historial) Listar() []RegistroHistorial {
	h.mu.Lock()
	defer h.mu.Unlock()
	copia := make([]RegistroHistorial, len(h.registros))
	copy(copia, h.registros)
	return copia
}

//go:embed web/index.html
var paginaWeb []byte

var historial = &Historial{}

type RespuestaError struct {
	Error string `json:"error"`
}

func responderJSON(w http.ResponseWriter, codigo int, valor any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	json.NewEncoder(w).Encode(valor)
}

const tamanoMaximo = 1 << 10 // 1 KB

// punteros para saber si un campo no se envió
type solicitud struct {
	Numero1  *float64 `json:"numero1"`
	Numero2  *float64 `json:"numero2"`
	Operador *string  `json:"operador"`
}

// leerOperacion lee el JSON y valida que esté completo
func leerOperacion(r *http.Request, w http.ResponseWriter) (Operacion, error) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, tamanoMaximo))
	dec.DisallowUnknownFields()

	var s solicitud
	if err := dec.Decode(&s); err != nil {
		return Operacion{}, fmt.Errorf("JSON inválido: %w", err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return Operacion{}, errors.New("JSON inválido: el cuerpo debe contener un solo objeto")
	}

	var faltan []string
	if s.Numero1 == nil {
		faltan = append(faltan, "numero1")
	}
	if s.Numero2 == nil {
		faltan = append(faltan, "numero2")
	}
	if s.Operador == nil {
		faltan = append(faltan, "operador")
	}
	if len(faltan) > 0 {
		return Operacion{}, fmt.Errorf("faltan campos en el JSON: %s", strings.Join(faltan, ", "))
	}
	return Operacion{Numero1: *s.Numero1, Numero2: *s.Numero2, Operador: *s.Operador}, nil
}

// manejarCalcular atiende las solicitudes a /calcular
func manejarCalcular(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderJSON(w, http.StatusMethodNotAllowed, RespuestaError{"use el método POST"})
		return
	}

	op, err := leerOperacion(r, w)
	if err != nil {
		responderJSON(w, http.StatusBadRequest, RespuestaError{err.Error()})
		return
	}

	if _, err := op.Calcular(); err != nil {
		responderJSON(w, http.StatusBadRequest, RespuestaError{err.Error()})
		return
	}

	historial.Agregar(op)
	log.Printf("%.2f %s %.2f = %.2f (cliente %s)", op.Numero1, op.Operador, op.Numero2, op.Resultado, r.RemoteAddr)
	responderJSON(w, http.StatusOK, op)
}

func manejarHistorial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderJSON(w, http.StatusMethodNotAllowed, RespuestaError{"use el método GET"})
		return
	}
	responderJSON(w, http.StatusOK, historial.Listar())
}

// manejarInicio muestra la página web
func manejarInicio(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(paginaWeb)
}

func nuevoRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", manejarInicio)
	mux.HandleFunc("/calcular", manejarCalcular)
	mux.HandleFunc("/historial", manejarHistorial)
	return mux
}

func main() {
	// puerto por defecto 8080
	puerto := os.Getenv("PUERTO")
	if puerto == "" {
		puerto = "8080"
	}
	direccion := ":" + puerto

	fmt.Printf("Calculadora escuchando en http://localhost%s\n", direccion)
	fmt.Println("  GET  /           página web de la calculadora")
	fmt.Println("  POST /calcular   {\"numero1\": 8, \"numero2\": 2, \"operador\": \"/\"}")
	fmt.Println("  GET  /historial")

	// tiempos máximos para no dejar conexiones colgadas
	servidor := &http.Server{
		Addr:              direccion,
		Handler:           nuevoRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Fatal(servidor.ListenAndServe())
}

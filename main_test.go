package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestCalcular(t *testing.T) {
	casos := []struct {
		nombre   string
		op       Operacion
		esperado float64
		err      error
	}{
		{"suma", Operacion{Numero1: 5, Numero2: 3, Operador: "+"}, 8, nil},
		{"resta", Operacion{Numero1: 10, Numero2: 4, Operador: "-"}, 6, nil},
		{"multiplicación", Operacion{Numero1: 6, Numero2: 7, Operador: "*"}, 42, nil},
		{"división", Operacion{Numero1: 8, Numero2: 2, Operador: "/"}, 4, nil},
		{"división decimal", Operacion{Numero1: 7, Numero2: 2, Operador: "/"}, 3.5, nil},
		{"división por cero", Operacion{Numero1: 8, Numero2: 0, Operador: "/"}, 0, ErrDivisionPorCero},
		{"operador inválido", Operacion{Numero1: 1, Numero2: 1, Operador: "%"}, 0, ErrOperadorInvalido},
		{"números negativos", Operacion{Numero1: -4, Numero2: 2.5, Operador: "*"}, -10, nil},
		{"resultado fuera de rango", Operacion{Numero1: 1e308, Numero2: 10, Operador: "*"}, 0, ErrFueraDeRango},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			res, err := c.op.Calcular()
			if err != c.err {
				t.Fatalf("error = %v, se esperaba %v", err, c.err)
			}
			if res != c.esperado {
				t.Fatalf("resultado = %v, se esperaba %v", res, c.esperado)
			}
		})
	}
}

// enviar hace una solicitud de prueba sin levantar el servidor
func enviar(metodo, ruta, cuerpo string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, ruta, bytes.NewBufferString(cuerpo))
	rec := httptest.NewRecorder()
	nuevoRouter().ServeHTTP(rec, req)
	return rec
}

func TestManejarCalcular(t *testing.T) {
	rec := enviar(http.MethodPost, "/calcular", `{"numero1": 8, "numero2": 2, "operador": "/"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("código = %d, cuerpo = %s", rec.Code, rec.Body)
	}
	var op Operacion
	json.Unmarshal(rec.Body.Bytes(), &op)
	if op.Resultado != 4 {
		t.Fatalf("resultado = %v, se esperaba 4", op.Resultado)
	}

	if rec := enviar(http.MethodPost, "/calcular", `{"numero1": 1, "numero2": 0, "operador": "/"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("división por cero: código = %d", rec.Code)
	}
	if rec := enviar(http.MethodPost, "/calcular", `no es json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("JSON inválido: código = %d", rec.Code)
	}
	if rec := enviar(http.MethodGet, "/calcular", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("método GET: código = %d", rec.Code)
	}
}

// 100 goroutines escribiendo a la vez (probar con go test -race)
func TestHistorialConcurrente(t *testing.T) {
	h := &Historial{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			h.Agregar(Operacion{Numero1: float64(n), Numero2: 1, Operador: "+", Resultado: float64(n + 1)})
		}(i)
	}
	wg.Wait()
	if got := len(h.Listar()); got != 100 {
		t.Fatalf("registros = %d, se esperaban 100", got)
	}
}

func TestManejarInicio(t *testing.T) {
	rec := enviar(http.MethodGet, "/", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Calculadora distribuida") {
		t.Fatalf("página web: código = %d", rec.Code)
	}
	if rec := enviar(http.MethodGet, "/no-existe", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("ruta inexistente: código = %d", rec.Code)
	}
}

func TestValidacionDeEntrada(t *testing.T) {
	casos := map[string]string{
		"cuerpo vacío":      ``,
		"campo faltante":    `{"numero1": 1, "operador": "+"}`,
		"sin números":       `{"operador": "+"}`,
		"tipo incorrecto":   `{"numero1": "a", "numero2": 1, "operador": "+"}`,
		"campo desconocido": `{"numero1": 1, "numero2": 2, "operador": "+", "extra": 1}`,
		"datos sobrantes":   `{"numero1": 1, "numero2": 2, "operador": "+"} basura`,
		"fuera de rango":    `{"numero1": 1e308, "numero2": 10, "operador": "*"}`,
		"cuerpo muy grande": `{"numero1": 1, "numero2": 2, "operador": "` + strings.Repeat("+", 2000) + `"}`,
	}
	for nombre, cuerpo := range casos {
		t.Run(nombre, func(t *testing.T) {
			rec := enviar(http.MethodPost, "/calcular", cuerpo)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("código = %d, cuerpo = %s", rec.Code, rec.Body)
			}
			var e RespuestaError
			if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Error == "" {
				t.Fatalf("la respuesta no trae un mensaje de error: %s", rec.Body)
			}
		})
	}
}

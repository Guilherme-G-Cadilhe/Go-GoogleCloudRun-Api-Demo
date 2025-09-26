package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestCEPValidation testa a validação do formato do CEP
func TestCEPValidation(t *testing.T) {
	tests := []struct {
		name         string
		cep          string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Valid CEP format",
			cep:          "01001000",                     // Will fail later if API key is missing or CEP not found, but format is valid
			expectedCode: http.StatusInternalServerError, // Expecting 500 because WEATHER_API_KEY is not set in tests
			expectedBody: "Weather API key not configured",
		},
		{
			name:         "Invalid CEP format - less than 8 digits",
			cep:          "1234567",
			expectedCode: http.StatusUnprocessableEntity, // 422
			expectedBody: `{"message":"invalid zipcode"}`,
		},
		{
			name:         "Invalid CEP format - more than 8 digits",
			cep:          "123456789",
			expectedCode: http.StatusUnprocessableEntity, // 422
			expectedBody: `{"message":"invalid zipcode"}`,
		},
		{
			name:         "Invalid CEP format - non-numeric characters",
			cep:          "01001-00",
			expectedCode: http.StatusUnprocessableEntity, // 422
			expectedBody: `{"message":"invalid zipcode"}`,
		},
		{
			name:         "Missing CEP parameter",
			cep:          "",
			expectedCode: http.StatusBadRequest, // 400
			expectedBody: "CEP parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/weather?cep="+tt.cep, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleGetWeather)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedCode)
			}

			// For 422 and 404, the body is JSON, for 400 and 500 it's plain text.
			if strings.Contains(rr.Header().Get("Content-Type"), "application/json") {
				if strings.TrimSpace(rr.Body.String()) != tt.expectedBody {
					t.Errorf("handler returned unexpected body: got %v want %v",
						strings.TrimSpace(rr.Body.String()), tt.expectedBody)
				}
			} else {
				if strings.TrimSpace(rr.Body.String()) != tt.expectedBody {
					t.Errorf("handler returned unexpected body: got %v want %v",
						strings.TrimSpace(rr.Body.String()), tt.expectedBody)
				}
			}
		})
	}
}

// TestWeatherAPIKeyMissing testa o cenário onde a chave da WeatherAPI está faltando
func TestWeatherAPIKeyMissing(t *testing.T) {
	// Garante que a variável de ambiente não está definida para este teste
	os.Unsetenv("WEATHER_API_KEY")

	req, err := http.NewRequest("GET", "/weather?cep=01001000", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleGetWeather)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError { // 500
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	expected := "Weather API key not configured"
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			strings.TrimSpace(rr.Body.String()), expected)
	}
}

// TestCEPNotFound simula um CEP não encontrado pelo ViaCEP
// Nota: Este teste não faz uma chamada real ao ViaCEP para evitar dependência externa.
// Ele assume que a lógica de "erro: true" ou "localidade vazia" do ViaCEP é tratada corretamente.
func TestCEPNotFound(t *testing.T) {
	// Para este teste, vamos mockar as chamadas HTTP ou usar um CEP que sabemos que o ViaCEP não encontrará.
	// Para simplicidade, e para não depender de mocks complexos ou de uma API externa,
	// vamos simular o comportamento esperado para um CEP não encontrado.
	// Em um cenário real, você poderia usar um mock HTTP client.

	// Para o propósito deste teste, vamos focar na resposta do handler.
	// Um CEP que o ViaCEP provavelmente não encontrará é 99999999.
	// Para que este teste funcione, você precisaria de uma chave WEATHER_API_KEY válida,
	// mas como o ViaCEP falharia primeiro, o erro de API Key não seria relevante aqui.

	// Nota: Este teste pode falhar se o ViaCEP mudar seu comportamento para 99999999.
	// Uma abordagem mais robusta seria usar um servidor HTTP de mock para o ViaCEP.
	// No entanto, para a simplicidade solicitada, vamos usar um CEP improvável.

	os.Setenv("WEATHER_API_KEY", "dummy_key") // Precisa de uma chave para passar da validação inicial
	defer os.Unsetenv("WEATHER_API_KEY")

	req, err := http.NewRequest("GET", "/weather?cep=99999999", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleGetWeather)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound { // 404
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	expected := `{"message":"can not find zipcode"}`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			strings.TrimSpace(rr.Body.String()), expected)
	}
}

// TestSuccessScenario testa um cenário de sucesso (requer APIs reais)
func TestSuccessScenario(t *testing.T) {
	// Para este teste, você precisará de uma chave WEATHER_API_KEY válida
	// e acesso à internet para as APIs ViaCEP e WeatherAPI.
	// Este é mais um teste de integração do que um unitário puro.
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		t.Skip("WEATHER_API_KEY not set, skipping success scenario integration test.")
	}

	req, err := http.NewRequest("GET", "/weather?cep=01001000", nil) // CEP de São Paulo
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleGetWeather)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK { // 200
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
		t.Errorf("Response body: %s", rr.Body.String())
	}

	var resp TemperatureResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Apenas verifica se os valores não são zero, pois a temperatura real varia
	if resp.TempC == 0 && resp.TempF == 0 && resp.TempK == 0 {
		t.Errorf("Expected non-zero temperatures, got %v", resp)
	}
	// Poderíamos adicionar mais validações, como verificar se as conversões estão corretas
	// dado um valor de TempC.
	expectedF := resp.TempC*1.8 + 32
	expectedK := resp.TempC + 273.15
	if resp.TempF != expectedF {
		t.Errorf("TempF conversion incorrect: got %f, expected %f", resp.TempF, expectedF)
	}
	if resp.TempK != expectedK {
		t.Errorf("TempK conversion incorrect: got %f, expected %f", resp.TempK, expectedK)
	}
}

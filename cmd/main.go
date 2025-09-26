package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

// ViaCEPResponse representa a estrutura da resposta da API ViaCEP
type ViaCEPResponse struct {
	Localidade string `json:"localidade"`
	Erro       bool   `json:"erro"`
}

// WeatherAPIResponse representa a estrutura da resposta da API WeatherAPI
type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// TemperatureResponse representa a estrutura da resposta final do nosso serviço
type TemperatureResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

// handleGetWeather lida com as requisições HTTP para obter o clima
func handleGetWeather(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cep := r.URL.Query().Get("cep")
	if cep == "" {
		http.Error(w, "CEP parameter is required", http.StatusBadRequest)
		return
	}

	// Validação do formato do CEP (8 dígitos numéricos)
	matched, _ := regexp.MatchString(`^\d{8}$`, cep)
	if !matched {
		w.WriteHeader(http.StatusUnprocessableEntity) // 422
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid zipcode"})
		return
	}

	// 1. Consultar ViaCEP para obter a cidade
	viaCEPURL := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	log.Printf("Consultando ViaCEP: %s", viaCEPURL)
	viaCEPResp, err := http.Get(viaCEPURL)
	if err != nil {
		log.Printf("Erro ao consultar ViaCEP: %v", err)
		http.Error(w, "Failed to get city information", http.StatusInternalServerError)
		return
	}
	defer viaCEPResp.Body.Close()

	if viaCEPResp.StatusCode != http.StatusOK {
		log.Printf("ViaCEP retornou status %d", viaCEPResp.StatusCode)
		http.Error(w, "Failed to get city information from ViaCEP", http.StatusInternalServerError)
		return
	}

	var viaCEPData ViaCEPResponse
	if err := json.NewDecoder(viaCEPResp.Body).Decode(&viaCEPData); err != nil {
		log.Printf("Erro ao decodificar resposta do ViaCEP: %v", err)
		http.Error(w, "Failed to parse city information", http.StatusInternalServerError)
		return
	}

	if viaCEPData.Erro || viaCEPData.Localidade == "" {
		w.WriteHeader(http.StatusNotFound) // 404
		json.NewEncoder(w).Encode(map[string]string{"message": "can not find zipcode"})
		return
	}

	cityName := viaCEPData.Localidade
	log.Printf("CEP %s encontrado. Cidade: %s", cep, cityName)

	// 2. Consultar WeatherAPI para obter a temperatura
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		log.Print("WEATHER_API_KEY não definida. Por favor, defina a variável de ambiente.")
		http.Error(w, "Weather API key not configured", http.StatusInternalServerError)
		return
	}

	encodedCityName := url.QueryEscape(cityName)
	weatherAPIURL := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s", weatherAPIKey, encodedCityName)
	log.Printf("Consultando WeatherAPI para %s", cityName)

	weatherResp, err := http.Get(weatherAPIURL)
	if err != nil {
		log.Printf("Erro ao consultar WeatherAPI: %v", err)
		http.Error(w, "Failed to get weather information", http.StatusInternalServerError)
		return
	}
	defer weatherResp.Body.Close()

	if weatherResp.StatusCode != http.StatusOK {
		// Tenta ler o corpo da resposta para ver a mensagem de erro da WeatherAPI
		bodyBytes, _ := io.ReadAll(weatherResp.Body)
		log.Printf("WeatherAPI retornou status %d. Corpo: %s", weatherResp.StatusCode, string(bodyBytes))

		var weatherError WeatherAPIResponse
		if err := json.Unmarshal(bodyBytes, &weatherError); err == nil && weatherError.Error.Code == 1006 {
			// Se a WeatherAPI não encontrar a localização, podemos considerar como CEP não encontrado para o usuário
			w.WriteHeader(http.StatusNotFound) // 404
			json.NewEncoder(w).Encode(map[string]string{"message": "can not find zipcode"})
			return
		}

		http.Error(w, "Failed to get weather information from WeatherAPI", http.StatusInternalServerError)
		return
	}

	var weatherData WeatherAPIResponse
	if err := json.NewDecoder(weatherResp.Body).Decode(&weatherData); err != nil {
		log.Printf("Erro ao decodificar resposta do WeatherAPI: %v", err)
		http.Error(w, "Failed to parse weather information", http.StatusInternalServerError)
		return
	}

	tempC := weatherData.Current.TempC
	log.Printf("Temperatura em %s: %.2f°C", cityName, tempC)

	// 3. Converter temperaturas
	tempF := tempC*1.8 + 32
	tempK := tempC + 273.15 // Usando 273.15 para maior precisão

	response := TemperatureResponse{
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/weather", handleGetWeather)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port for local development
	}

	log.Printf("Servidor iniciado na porta :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

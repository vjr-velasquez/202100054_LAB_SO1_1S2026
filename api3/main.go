package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        string `json:"vm"`
	Carnet    string `json:"carnet"`
}

type ConnectionResponse struct {
	APIName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

// Variables para almacenar las URLs de las APIs
var api1URL string
var api2URL string

func health(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "UP",
		Message:   "API3 is Ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		VM:        "VM2",
		Carnet:    "202100054",
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println("Error generando la respuesta: ", err)
	}
}

// callAPI1 comprueba si API1 está funcionando en VM1.
func callAPI1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	apiResponse, err := http.Get(api1URL + "/health")

	result := ConnectionResponse{
		APIName:    "API1",
		Message:    "ERROR: The API1 located on the VM1 is not working",
		Connection: false,
		Carnet:     "202100054",
	}

	if err == nil {
		defer apiResponse.Body.Close()

		var api1Health HealthResponse

		err = json.NewDecoder(apiResponse.Body).Decode(&api1Health)

		if err == nil &&
			apiResponse.StatusCode == http.StatusOK &&
			api1Health.Status == "UP" {

			result.Message = "The API1 located on the VM1 is working"
			result.Connection = true
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Println("Error generando la respuesta:", err)
	}
}

// callAPI2 comprueba si API2 está funcionando en VM1.
func callAPI2(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	apiResponse, err := http.Get(api2URL + "/health")

	result := ConnectionResponse{
		APIName:    "API2",
		Message:    "ERROR: The API2 located on the VM1 is not working",
		Connection: false,
		Carnet:     "202100054",
	}

	if err == nil {
		defer apiResponse.Body.Close()

		var api2Health HealthResponse
		err = json.NewDecoder(apiResponse.Body).Decode(&api2Health)

		if err == nil &&
			apiResponse.StatusCode == http.StatusOK &&
			api2Health.Status == "UP" {

			result.Message = "The API2 located on the VM1 is working"
			result.Connection = true
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Println("Error generando la respuesta:", err)
	}
}

func main() {

	api1URL = os.Getenv("API1_URL")
	api2URL = os.Getenv("API2_URL")

	if api1URL == "" {
		log.Fatal("La variable de entorno 'API1_URL' no está configurada")
	}

	if api2URL == "" {
		log.Fatal("La variable de entorno 'API2_URL' no está configurada")
	}

	http.HandleFunc("/health", health)
	http.HandleFunc("/api3/202100054/call-api1", callAPI1)
	http.HandleFunc("/api3/202100054/call-api2", callAPI2)

	log.Println("API3 iniciada en el puerto 8083")

	if err := http.ListenAndServe(":8083", nil); err != nil {
		log.Fatal("No se pudo iniciar API3: ", err)
	}
}

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// HealthResponse define los campos de la respuesta JSON.
type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        string `json:"VM"`
	Carnet    string `json:"carnet"`
}

// ConnectionResponse define la respuesta de una llamada entre APIs.
type ConnectionResponse struct {
	APIName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

// Variables para almacenar las URLs de las APIs.
var api1URL string
var api3URL string

// health responde cuando se visita /health.
func health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "UP",
		Message:   "API2 is Ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		VM:        "VM1",
		Carnet:    "202100054",
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error generando la respuesta", http.StatusInternalServerError)
	}
}

// hacemos el callAPI1 que comprueba si api1 esta funcionando
func callAPI1(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	// API2 realiza una solicitud al endpoint /health de API1.
	apiResponse, err := http.Get(api1URL + "/health")

	// Preparamos inicialmente una respuesta
	result := ConnectionResponse{
		APIName:    "API1",
		Message:    "ERROR: The API1 located on the VM1 is not working",
		Connection: false,
		Carnet:     "202100054",
	}

	// Si API1 respondió, revisamos el contenido de su JSON.
	if err == nil {
		defer apiResponse.Body.Close()

		var api1Health HealthResponse

		err = json.NewDecoder(apiResponse.Body).Decode(&api1Health)

		// La conexión es exitosa solamente si API1 devuelve status UP.
		if err == nil && apiResponse.StatusCode == http.StatusOK &&
			api1Health.Status == "UP" {

			result.Message = "The API1 located on the VM1 is working"
			result.Connection = true
		}
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		log.Println("Error generando la respuesta de conexión a API1: ", err)
	}

}

// callAPI3 comprueba si API3 está funcionando en VM2.
func callAPI3(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	apiResponse, err := http.Get(api3URL + "/health")

	result := ConnectionResponse{
		APIName:    "API3",
		Message:    "ERROR: The API3 located on the VM2 is not working",
		Connection: false,
		Carnet:     "202100054",
	}

	if err == nil {
		defer apiResponse.Body.Close()

		var api3Health HealthResponse
		err = json.NewDecoder(apiResponse.Body).Decode(&api3Health)

		if err == nil &&
			apiResponse.StatusCode == http.StatusOK &&
			api3Health.Status == "UP" {

			result.Message = "The API3 located on the VM2 is working"
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
	api3URL = os.Getenv("API3_URL")

	if api1URL == "" {
		log.Fatal("La variable API1_URL no está definida")
	}

	if api3URL == "" {
		log.Fatal("La variable API3_URL no está definida")
	}

	http.HandleFunc("/health", health)
	http.HandleFunc("/api2/202100054/call-api1", callAPI1)
	http.HandleFunc("/api2/202100054/call-api3", callAPI3)

	log.Println("API2 iniciada en el puerto 8082")

	err := http.ListenAndServe(":8082", nil)
	if err != nil {
		log.Fatal("No se pudo iniciar API2: ", err)
	}
}

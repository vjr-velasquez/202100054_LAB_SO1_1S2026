package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// HealthResponse , representa la estructura del formato JSON que devolvera /health

type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        string `json:"vm"`
	Carnet    string `json:"carnet"`
}

// ConnectionResponse define la respuesta de una llamada entre APIs.
type ConnectionResponse struct {
	APIName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

// Definimos las variables de entorno para las URLs de las APIs
var api2URL string
var api3URL string

// Hacemos una funcion que responde cuando alguien visita /health

func health(w http.ResponseWriter, r *http.Request) {

	// Solo se aceptan las solicitudes GET

	if r.Method != "GET" {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Construimos la estructura de la respuesta
	response := HealthResponse{
		Status:    "UP",
		Message:   "API1 is Ready",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		VM:        "VM1",
		Carnet:    "202100054",
	}

	// Indicamos que la respuesta ser JSON
	w.Header().Set("Content-Type", "application/json")

	// Convertimos la estructura de GO a JSON y se envia

	err := json.NewEncoder(w).Encode(response)

	if err != nil {
		http.Error(w, "Error generando la respuesta JSON", http.StatusInternalServerError)
	}
}

// callAPI2 comprueba si API2 está funcionando.
func callAPI2(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// API1 consulta el endpoint /health de API2.
	apiResponse, err := http.Get(api2URL + "/health")

	// Por defecto suponemos que la conexión falló.
	result := ConnectionResponse{
		APIName:    "API2",
		Message:    "ERROR: The API2 located on the VM1 is not working",
		Connection: false,
		Carnet:     "202100054",
	}

	// Si API2 respondió, analizamos su JSON.
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

	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		log.Println("Error generando la respuesta:", err)
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

	api2URL = os.Getenv("API2_URL")
	api3URL = os.Getenv("API3_URL")

	if api2URL == "" {
		log.Fatal("La variable de API2_URL no esta definida")
	}

	if api3URL == "" {
		log.Fatal("La variable de API3_URL no esta definida")
	}

	// Cuando alguien visite /health, se ejecutara la funcion health
	http.HandleFunc("/health", health)
	http.HandleFunc("/api1/202100054/call-api2", callAPI2)
	http.HandleFunc("/api1/202100054/call-api3", callAPI3)

	log.Println(" API1 iniciada en el puerto 8081")

	// Iniciamos el servidor

	err := http.ListenAndServe(":8081", nil)

	if err != nil {
		log.Fatal("No se pudo iniciar API1 ", err)
	}
}

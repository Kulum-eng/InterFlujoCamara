package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"

)

type RequestBody struct {
	Base64 string `json:"base64"`
}

func main() {

	rabbitMQUser := "ale"
	rabbitMQPass := "ale123"
	rabbitMQHost := "ec2-54-167-194-141.compute-1.amazonaws.com"
	rabbitMQPort := "5672"
	queueName := "imagenes"

	rabbitMQURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitMQUser, rabbitMQPass, rabbitMQHost, rabbitMQPort)

	http.HandleFunc("/camara", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var body RequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
			return
		}

		if body.Base64 == "" {
			http.Error(w, "Campo 'base64' requerido", http.StatusBadRequest)
			return
		}

		conn, err := amqp.Dial(rabbitMQURL)
		if err != nil {
			log.Printf("Error conectando a RabbitMQ: %v", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		ch, err := conn.Channel()
		if err != nil {
			log.Printf("Error al abrir canal: %v", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}
		defer ch.Close()

		_, err = ch.QueueDeclare(
			queueName,
			true,  // durable
			false, // autoDelete
			false, // exclusive
			false, // noWait
			nil,   // args
		)
		if err != nil {
			log.Printf("Error al declarar cola: %v", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		err = ch.Publish(
			"",        // exchange
			queueName, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType:  "text/plain",
				Body:         []byte(body.Base64),
				DeliveryMode: amqp.Persistent,
			},
		)
		if err != nil {
			log.Printf("Error al publicar: %v", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Imagen enviada a RabbitMQ",
		})
	})

	port := ":8443"
	log.Printf("Servidor InterFlujo escuchando en %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

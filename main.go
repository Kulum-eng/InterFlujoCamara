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
	Token  string `json:"token"`
}

func main() {
	rabbitMQUser := "ale"
	rabbitMQPass := "ale123"
	rabbitMQHost := "ec2-54-167-194-141.compute-1.amazonaws.com"
	rabbitMQPort := "5672"
	queueName := "imagenes"

	rabbitMQURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitMQUser, rabbitMQPass, rabbitMQHost, rabbitMQPort)

	http.HandleFunc("/camara", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

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

		messageBody := fmt.Sprintf(`{"base64": "%s", "token": "%s"}`, body.Base64, body.Token)
		err = ch.Publish(
			"",        // exchange
			queueName, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         []byte(messageBody),
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

	go func() {
		conn, err := amqp.Dial(rabbitMQURL)
		if err != nil {
			log.Fatalf("Error conectando a RabbitMQ: %v", err)
		}
		defer conn.Close()

		ch, err := conn.Channel()
		if err != nil {
			log.Fatalf("Error al abrir canal: %v", err)
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
			log.Fatalf("Error al declarar cola: %v", err)
		}

		msgs, err := ch.Consume(
			queueName,
			"",    // consumer
			true,  // autoAck
			false, // exclusive
			false, // noLocal
			false, // noWait
			nil,   // args
		)
		if err != nil {
			log.Fatalf("Error al consumir mensajes: %v", err)
		}

		for msg := range msgs {
			var body RequestBody
			if err := json.Unmarshal(msg.Body, &body); err != nil {
				log.Printf("Error al decodificar mensaje: %v", err)
				continue
			}

			log.Printf("Mensaje recibido - Base64: %s, Token: %s", body.Base64, body.Token)
			
		}
	}()

	port := ":8443"
	log.Printf("Servidor InterFlujo escuchando en %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

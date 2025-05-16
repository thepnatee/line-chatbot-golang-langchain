package main

import (
	"log"
	"net/http"
	"os"

	"line-chatbot-golang-langchain/handler"
	"line-chatbot-golang-langchain/utils"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	err = utils.InitMongo()
	if err != nil {
		log.Fatal("Mongo init error:", err)
	}
	defer utils.CloseMongo()

	http.HandleFunc("/init-disc-vectors", handler.InitDiscVectorsHandler)
	http.HandleFunc("/submit-answer", handler.AnswerSubmissionHandler)

	http.HandleFunc("/callback", handler.LineWebhookHandler)

	log.Println("📌 Available Routes:")
	log.Println("✅ POST /callback           → LINE webhook endpoint")
	log.Println("✅ POST /submit-answer      →Answer Submission")
	log.Println("✅ GET  /init-disc-vectors  → Initialize DISC embeddings")

	port := os.Getenv("PORT")
	if port == "" {
		port = "5001"
	}
	log.Println("Server started at port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

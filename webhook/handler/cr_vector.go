package handler

import (
	"encoding/json"
	"fmt"
	"line-chatbot-golang-langchain/models"
	"line-chatbot-golang-langchain/utils"
	"log"
	"net/http"
	"strings"
)

// InitDiscVectorsHandler เรียกสร้างเวกเตอร์ DISC และ Index
func InitDiscVectorsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔧 InitDiscVectorsHandler called - starting async vector initialization...")
	go utils.InsertVectors() // async ไม่ block user
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "✅ กำลังสร้าง DISC embeddings และ index แล้ว...")
}

func AnswerSubmissionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("📥 Received request to /submit-answer")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Groupid")
	w.Header().Set("Access-Control-Max-Age", "86400") // cache preflight 24h

	if r.Method == http.MethodOptions {
		log.Println("🔁 OPTIONS request received (CORS Preflight)")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Println("🚫 Invalid HTTP Method:", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID := r.Header.Get("groupid")
	idToken := r.Header.Get("Authorization")

	if idToken == "" {
		log.Println("🚫 Missing Authorization header")
		http.Error(w, "Missing groupId or Authorization header", http.StatusBadRequest)
		return
	}

	log.Println("🛂 Extracted Headers - groupID:", groupID, "idToken:", idToken)

	var req models.AnswerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("🚫 Failed to decode request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Answers) == 0 {
		log.Println("🚫 No answers submitted")
		http.Error(w, "Missing answers", http.StatusBadRequest)
		return
	}

	var indexedAnswers []string
	for i, answer := range req.Answers {
		if len(answer) == 0 {
			log.Printf("⚠️ Skipping empty answer at index %d\n", i)
			continue
		}
		indexed := fmt.Sprintf("%d.%s", i+1, answer)
		indexedAnswers = append(indexedAnswers, indexed)
	}
	formattedAnswers := strings.Join(indexedAnswers, ", ")

	log.Println("✅ Formatted answers:", formattedAnswers)

	profile, err := utils.GetProfileByIDToken(idToken)
	if err != nil || profile["sub"] == nil {
		log.Println("🚫 Invalid LINE ID token or missing profile")
		http.Error(w, "Invalid LINE ID Token", http.StatusUnauthorized)
		return
	}

	userID := profile["sub"].(string)
	log.Println("👤 LINE User ID:", userID)

	prompt := formattedAnswers
	log.Println("📤 Sending prompt to Gemini:", prompt)

	jsonString, err := utils.VectorSearchQueryGemini(prompt, true)
	if err != nil {
		log.Println("❌ Gemini vector search failed:", err)
		http.Error(w, "Gemini search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("📥 Gemini JSON Response:", jsonString)

	if err := json.Unmarshal([]byte(jsonString), &models.AiResult); err != nil {
		log.Println("❌ Failed to parse AI response:", err)
		http.Error(w, "Failed to parse AI response", http.StatusInternalServerError)
		return
	}

	userAnswer := map[string]interface{}{
		"userId":      userID,
		"groupId":     groupID,
		"model":       models.AiResult.Model,
		"description": models.AiResult.Description,
		"answers":     req.Answers,
	}

	log.Println("📝 Saving user answer to MongoDB:", userAnswer)

	if err := utils.UpsertAnswersByUserID(userID, groupID, userAnswer); err != nil {
		log.Println("❌ Failed to save user answer:", err)
		http.Error(w, "Mongo save failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("✅ User answer saved successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User answer saved successfully",
		"data":    userAnswer,
	})
	if err != nil {
		log.Println("⚠️ Failed to encode response:", err)
	}
}

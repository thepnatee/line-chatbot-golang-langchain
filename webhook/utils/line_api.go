package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"line-chatbot-golang-langchain/models"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func VerifySignature(signature string, body []byte) bool {
	secret := os.Getenv("LINE_CHANNEL_SECRET")
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	expectedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	valid := hmac.Equal([]byte(expectedSignature), []byte(signature))
	if !valid {
		log.Println("❌ Signature verification failed")
	}
	return valid
}

func ReplyMessage(replyToken string, messages []interface{}) error {
	body := models.MessageBody{
		ReplyToken: replyToken,
		Messages:   messages,
	}

	log.Println("📤 กำลังส่งข้อความกลับไปยัง LINE Messaging API...")
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Println("❌ JSON marshal error:", err)
		return err
	}

	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/reply", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Println("❌ Failed to create request:", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("❌ Request to LINE Messaging API failed:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("❌ LINE Messaging API returned error. Status: %d Body: %s", resp.StatusCode, string(body))
		return errors.New("line API error")
	}

	log.Println("✅ ส่งข้อความสำเร็จแล้ว")
	return nil
}

func GetProfileByIDToken(idToken string) (map[string]interface{}, error) {
	apiURL := os.Getenv("LINE_ENDPOINT_API_VERIFY")
	clientID := os.Getenv("LINE_LIFF_CHANNEL_ID")

	if apiURL == "" || clientID == "" {
		log.Println("❌ Environment variables LINE_ENDPOINT_API_VERIFY or LINE_LIFF_CHANNEL_ID are missing")
		return nil, errors.New("missing LINE_ENDPOINT_API_VERIFY or LINE_LIFF_CHANNEL_ID in .env")
	}

	log.Println("🔍 กำลังตรวจสอบ ID Token กับ LINE API...")

	data := url.Values{}
	data.Set("id_token", idToken)
	data.Set("client_id", clientID)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		log.Println("❌ Failed to create request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("❌ Request to LINE verify API failed:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("❌ LINE verify API responded with status %d: %s", resp.StatusCode, string(body))
		return nil, errors.New("LINE verify API failed: " + string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("❌ Failed to read response body:", err)
		return nil, err
	}

	profile, err := ParseJSONToMap(body)
	if err != nil {
		log.Println("❌ Failed to parse profile JSON:", err)
		return nil, err
	}

	log.Println("✅ รับข้อมูลโปรไฟล์จาก LINE สำเร็จ:", profile)
	return profile, nil
}

func ParseJSONToMap(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		log.Println("❌ JSON Unmarshal error:", err)
	}
	return result, err
}

package utils

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func AskGemini(prompt string) (string, error) {
	log.Println("📨 เรียกใช้งาน AskGemini ด้วย prompt:")
	log.Println(prompt)

	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Println("❌ ไม่สามารถสร้าง Gemini client ได้:", err)
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer func() {
		err := client.Close()
		if err != nil {
			log.Println("⚠️ Error closing Gemini client:", err)
		} else {
			log.Println("🔒 Gemini client ปิดการเชื่อมต่อแล้ว")
		}
	}()

	log.Println("🧠 เรียกใช้ Gemini model: gemini-2.0-flash")
	model := client.GenerativeModel("gemini-2.0-flash")

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Println("❌ เกิดข้อผิดพลาดในการสร้างเนื้อหาด้วย Gemini:", err)
		return "", fmt.Errorf("Gemini content generation failed: %w", err)
	}

	log.Println("✅ รับผลลัพธ์จาก Gemini แล้ว กำลังตรวจสอบเนื้อหา...")

	if len(resp.Candidates) > 0 {
		if len(resp.Candidates[0].Content.Parts) > 0 {
			if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
				result := string(text)
				log.Println("📝 เนื้อหาที่ได้จาก Gemini:", result)
				return result, nil
			}
		}
	}

	log.Println("⚠️ ไม่พบเนื้อหาจาก Gemini ที่สามารถแสดงได้")
	return "", fmt.Errorf("no content returned from Gemini")
}

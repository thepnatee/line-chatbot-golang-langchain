package utils

import (
	"context"
	"fmt"
	"io"
	"line-chatbot-golang-langchain/models"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings/huggingface"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/mongovector"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func InsertVectors() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("🔗 Connecting to MongoDB...")
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		log.Fatal("❌ Mongo connect error:", err)
	}
	defer func() {
		log.Println("🔌 Disconnecting MongoDB...")
		client.Disconnect(ctx)
	}()

	fmt.Println("Key:", os.Getenv("HUGGINGFACEHUB_API_TOKEN"))

	db := client.Database("developer")
	collection := db.Collection("disc_embeddings")

	// ดาวน์โหลดข้อมูลหน้าเว็บและเตรียมไฟล์
	filename := "landing-page.html"
	DownloadReport(filename)

	log.Println("📄 Processing HTML file into chunks...")
	docs := ProcessFile(filename)

	// โหลด environment variables (.env)
	if err := godotenv.Load(); err != nil {
		log.Printf("❌ .env file not found")
		return
	}

	log.Println("🧠 Initializing embedding model (HuggingFace)...")
	embedder, err := huggingface.NewHuggingface(
		huggingface.WithModel("sentence-transformers/all-mpnet-base-v2"),
		huggingface.WithTask("feature-extraction"))
	if err != nil {
		log.Printf("❌ Failed to create an embedder: %v", err)
		return
	}

	// เตรียม MongoDB vector store
	store := mongovector.New(collection, embedder, mongovector.WithPath("embedding"))

	log.Println("📦 Inserting documents into vector store (MongoDB Atlas)...")
	result, err := store.AddDocuments(context.Background(), docs)
	if err != nil {
		log.Printf("❌ Failed to insert documents: %v", err)
		return
	}
	log.Printf("✅ Successfully inserted %v documents into Atlas\n", len(result))

	// สร้าง vector index ด้วย Go SDK
	err = CreateVectorIndexWithSDK(collection)
	if err != nil {
		log.Println("❌ Failed to create Atlas vector index:", err)
	} else {
		log.Println("✅ Vector Index created or already exists")
	}
}

func CreateVectorIndexWithSDK(coll *mongo.Collection) error {
	ctx := context.Background()
	indexName := "vector_index"

	opts := options.SearchIndexes().
		SetName(indexName).
		SetType("vectorSearch")

	// กำหนด schema สำหรับ vector index
	indexModel := mongo.SearchIndexModel{
		Definition: models.VectorDefinition{
			Fields: []models.VectorDefinitionField{{
				Type:          "vector",
				Path:          "embedding",
				NumDimensions: 768,
				Similarity:    "cosine",
			}},
		},
		Options: opts,
	}

	log.Println("⚙️ Creating Atlas vector index via Go SDK...")
	searchIndexName, err := coll.SearchIndexes().CreateOne(ctx, indexModel)
	if err != nil {
		return err
	}

	log.Println("🔍 Polling to confirm successful index creation...")
	searchIndexes := coll.SearchIndexes()
	var doc bson.Raw
	for doc == nil {
		cursor, err := searchIndexes.List(ctx, options.SearchIndexes().SetName(searchIndexName))
		if err != nil {
			log.Printf("❌ Failed to list search indexes: %v", err)
		}
		if !cursor.Next(ctx) {
			break
		}
		name := cursor.Current.Lookup("name").StringValue()
		queryable := cursor.Current.Lookup("queryable").Boolean()
		if name == searchIndexName && queryable {
			doc = cursor.Current
		} else {
			time.Sleep(5 * time.Second)
		}
	}
	log.Println("✅ Index confirmed: " + searchIndexName)
	return nil
}

func DownloadReport(filename string) {
	// ถ้าไฟล์มีอยู่แล้ว ให้ข้าม
	if _, err := os.Stat(filename); err == nil {
		log.Println("📁 File already exists:", filename)
		return
	}

	const url = "https://www.baseplayhouse.co/blog/what-is-disc"
	log.Println("🌐 Downloading", url, "→", filename)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("❌ Failed to download the report: %v", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("❌ Failed to create file: %v", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		log.Fatalf("❌ Failed to write file: %v", err)
	}

	log.Println("✅ File downloaded successfully:", filename)
}

func ProcessFile(filename string) []schema.Document {
	ctx := context.Background()

	log.Println("📂 Opening HTML file:", filename)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("❌ Failed to open file: %v", err)
	}
	defer f.Close()

	html := documentloaders.NewHTML(f)
	split := textsplitter.NewRecursiveCharacter()
	split.ChunkSize = 400
	split.ChunkOverlap = 20

	log.Println("✂️ Splitting content into chunks...")
	docs, err := html.LoadAndSplit(ctx, split)
	if err != nil {
		log.Fatalf("❌ Failed to chunk HTML: %v", err)
	}
	log.Printf("✅ Successfully chunked into %v documents.\n", len(docs))
	return docs
}

func GetQueryResults(query string) []schema.Document {
	coll := client.Database("developer").Collection("disc_embeddings")

	log.Println("🔍 Performing vector similarity search for query:", query)
	embedder, err := huggingface.NewHuggingface(
		huggingface.WithModel("sentence-transformers/all-mpnet-base-v2"),
		huggingface.WithTask("feature-extraction"))
	if err != nil {
		log.Fatalf("❌ Failed to create embedder: %v", err)
	}

	store := mongovector.New(coll, embedder, mongovector.WithPath("embedding"))
	docs, err := store.SimilaritySearch(context.Background(), query, 5)
	if err != nil {
		log.Fatalf("❌ Similarity search failed: %v", err)
	}

	log.Printf("✅ Found %d similar documents.\n", len(docs))
	return docs
}

func VectorSearchQueryGemini(userText string, checkJSON bool) (string, error) {
	documents := GetQueryResults(userText)

	var textDocuments strings.Builder
	for _, doc := range documents {
		textDocuments.WriteString(doc.PageContent)
		textDocuments.WriteString("\n\n")
	}

	// สร้าง prompt สำหรับ Gemini
	prompt := fmt.Sprintf(`
	คุณคือผู้เชี่ยวชาญด้าน DISC Model ซึ่งแบ่งบุคลิกภาพออกเป็น 4 กลุ่ม คือ D (Dominance), I (Influence), S (Steadiness), C (Conscientiousness)
	พิจารณาบุคลิกภาพต่อไปนี้:
	"%s"

	และจากข้อมูล DISC ด้านล่าง:
	%s

	ช่วยระบุว่าบุคคลนี้น่าจะตรงกับ DISC ประเภทใดมากที่สุด และให้คำอธิบายอย่างกระชับ พร้อมตอบในรูปแบบ JSON:
	{
	"model": "ประเภท DISC ที่เหมาะสม",
	"description": "คำอธิบายเหตุผลที่เลือกประเภทนี้"
	}
`, userText, textDocuments.String())

	// เรียก API Gemini
	answer, err := AskGemini(prompt)
	if err != nil {
		log.Printf("❌ Gemini error: %v", err)
		return "", err
	}

	// ล้าง markdown JSON ถ้ามี
	if checkJSON {
		answer = strings.ReplaceAll(answer, "```json", "")
		answer = strings.ReplaceAll(answer, "```", "")
		answer = strings.TrimSpace(answer)
	}

	return answer, nil
}

func GetAllUsersInGroup(groupID string) ([]bson.M, error) {
	filter := bson.M{"groupId": groupID}
	cursor, err := groupCol.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		return nil, err
	}
	return results, nil
}

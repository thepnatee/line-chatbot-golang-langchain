# 🤖 LINE Chatbot with DISC Personality Model (LangChain + Gemini + MongoDB)

A smart chatbot integrated with LINE Messaging API that uses Google Gemini AI and MongoDB Atlas Vector Search to analyze user personality using the **DISC model**.

> Built with Go, LangChainGo, and Gemini API. 🚀

---

## 🧠 Features

- LINE Bot integrated via Webhook
- Gemini AI + Vector Search to analyze DISC type
- Auto-greeting when bot joins or user joins a group
- Personalized messages with LINE mentions
- MongoDB used for vector storage and user data persistence

---

## 🔧 Tech Stack

- **Go (Golang)**
- **MongoDB Atlas** with Vector Index
- **LangChainGo** + Huggingface Embedding + Gemini (via REST)
- **LINE Messaging API**
- **ENV & Config**: `godotenv`

---

## 🛠️ API Endpoints

| Method | Endpoint               | Description                    |
|--------|------------------------|--------------------------------|
| POST   | `/callback`            | LINE Webhook for receiving events |
| POST   | `/submit-answer`       | User submits answers to DISC test |
| GET    | `/init-disc-vectors`   | Initializes DISC embeddings into MongoDB |

---

## 🚀 Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/your-name/line-disc-chatbot.git
cd line-disc-chatbot
```

### 2 Create .env
```env
LINE_CHANNEL_SECRET=xxx
LINE_CHANNEL_ACCESS_TOKEN=xxx
LINE_LIFF_DISC=https://liff.line.me/...
GEMINI_API_KEY=your-api-key
MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net
```

### 3 Initialize MongoDB
```bash
go get . && go mod tidy
```

### 4. Run the bot

```bash
go run .
```

---

## 💡 How DISC Analysis Works
- User answers 20 questions in LIFF frontend
- Bot formats answers to a prompt
- Gemini AI + Vector Search retrieves the most relevant DISC personality
- Response saved & returned to user with friendly message
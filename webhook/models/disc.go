package models

type MessageBody struct {
	ReplyToken string      `json:"replyToken"`
	Messages   interface{} `json:"messages"`
}

type AnswerRequest struct {
	Answers []string `json:"answers"`
}

type VectorDefinitionField struct {
	Type          string `bson:"type"`
	Path          string `bson:"path"`
	NumDimensions int    `bson:"numDimensions"`
	Similarity    string `bson:"similarity"`
}

type VectorDefinition struct {
	Fields []VectorDefinitionField `bson:"fields"`
}

var AiResult struct {
	Model       string `json:"model"`
	Description string `json:"description"`
}

package utils

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var client *mongo.Client
var groupCol *mongo.Collection
var ctx context.Context

func InitMongo() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("🛠️ Connecting to MongoDB...")

	opt := options.Client().ApplyURI(os.Getenv("MONGO_URI"))
	client, err = mongo.Connect(opt)
	if err != nil {
		log.Println("❌ Failed to connect to MongoDB:", err)
		return err
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Println("❌ MongoDB ping failed:", err)
		return err
	}

	groupCol = client.Database("developer").Collection("groups")
	log.Println("✅ MongoDB connected and 'groups' collection ready.")
	return nil
}

func UpsertGroup(groupID string) error {
	filter := bson.M{"groupId": groupID}
	update := bson.M{
		"$set": bson.M{
			"groupId":   groupID,
			"updatedAt": time.Now(),
		},
	}

	log.Println("📦 Upserting group:", groupID)
	opts := options.UpdateOne().SetUpsert(true)
	_, err := groupCol.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		log.Println("❌ UpsertGroup error:", err)
	} else {
		log.Println("✅ UpsertGroup success:", groupID)
	}
	return err
}

func DeleteGroup(groupID string) error {
	log.Println("🗑️ Deleting group:", groupID)
	_, err := groupCol.DeleteOne(context.Background(), bson.M{"groupId": groupID})
	if err != nil {
		log.Println("❌ DeleteGroup error:", err)
	} else {
		log.Println("✅ Group deleted:", groupID)
	}
	return err
}

func UpsertAnswersByUserID(userID, groupID string, data map[string]interface{}) error {
	filter := bson.M{"userId": userID, "groupId": groupID}
	data["updatedAt"] = time.Now()

	update := bson.M{"$set": data}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	log.Printf("📨 Upserting answers for userID=%s, groupID=%s\n", userID, groupID)
	var updated bson.M
	err := groupCol.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Println("❌ UpsertAnswers error:", err)
		return err
	}

	log.Println("✅ Answers upserted for:", userID)
	return nil
}

func GetAnswersByUserID(userID, groupID string) (map[string]interface{}, error) {
	filter := bson.M{"userId": userID}
	if groupID != "" {
		filter["groupId"] = groupID
	}

	var result map[string]interface{}
	err := groupCol.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("⚠️ No document found for userId: %s, groupId: %s", userID, groupID)
			return nil, nil
		}
		log.Println("❌ GetAnswersByUserID error:", err)
		return nil, err
	}

	return result, nil
}

func CloseMongo() {
	if client != nil {
		log.Println("🔌 Closing MongoDB connection...")
		if err := client.Disconnect(ctx); err != nil {
			log.Println("❌ Error closing Mongo:", err)
		} else {
			log.Println("✅ MongoDB connection closed.")
		}
	}
}

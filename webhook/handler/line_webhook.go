package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"line-chatbot-golang-langchain/utils"
)

func LineWebhookHandler(w http.ResponseWriter, req *http.Request) {
	signature := req.Header.Get("X-Line-Signature")
	if signature == "" {
		http.Error(w, "Missing Signature", http.StatusUnauthorized)
		log.Println("🚫 Missing Signature header")
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		log.Println("🚫 Failed to read body:", err)
		return
	}

	if !utils.VerifySignature(signature, body) {
		http.Error(w, "Invalid Signature", http.StatusUnauthorized)
		log.Println("🚫 Invalid LINE signature")
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		log.Println("🚫 Invalid JSON payload:", err)
		return
	}

	events := payload["events"].([]interface{})
	for _, e := range events {
		event := e.(map[string]interface{})
		eventType := event["type"].(string)
		log.Println("📩 Handling LINE event:", eventType)

		switch eventType {
		case "join":
			handleJoinEvent(event)
		case "memberJoined":
			handleMemberJoinedEvent(event)
		case "message":
			handleMessageEvent(event, w)
		case "leave":
			handleLeaveEvent(event)
		}
	}

	w.WriteHeader(http.StatusOK)
	log.Println("✅ Webhook processed successfully")
}

func handleJoinEvent(event map[string]interface{}) {
	source := event["source"].(map[string]interface{})
	groupID := source["groupId"].(string)
	replyToken := event["replyToken"].(string)

	log.Println("👥 Bot joined group:", groupID)

	liffURL := os.Getenv("LINE_LIFF_DISC") + "?groupId=" + groupID

	// if err := utils.UpsertGroup(groupID); err != nil {
	// 	log.Println("❌ Failed to upsert group:", err)
	// 	return
	// }

	message := map[string]interface{}{
		"type": "text",
		"text": "สวัสดีทุกค๊นน มารวมกันทำแบบสอบถามกันเถอะ \r\n หากต้องการเริ่มทำแบบสอบถามใหม่ \n เพียง tag ชื่อ @disc ได้เลย ",
		"quickReply": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"type": "action",
					"action": map[string]interface{}{
						"type":  "uri",
						"label": "เริ่มทำแบบทดสอบ",
						"uri":   liffURL,
					},
				},
				map[string]interface{}{
					"type": "action",
					"action": map[string]interface{}{
						"type":  "message",
						"label": "Type",
						"text":  "Type",
					},
				},
			},
		},
	}

	utils.ReplyMessage(replyToken, []interface{}{message})
	log.Println("✅ Sent join message to group:", groupID)
}

func handleMemberJoinedEvent(event map[string]interface{}) {
	replyToken := event["replyToken"].(string)
	groupID := event["source"].(map[string]interface{})["groupId"].(string)
	liffURL := os.Getenv("LINE_LIFF_DISC") + "?groupId=" + groupID
	members := event["joined"].(map[string]interface{})["members"].([]interface{})

	for _, m := range members {
		member := m.(map[string]interface{})
		if member["type"] != "user" {
			continue
		}
		userID := member["userId"].(string)

		message := map[string]interface{}{
			"type": "textV2",
			"text": "สวัสดีคุณ {user1}! ยินดีต้อนรับ \n ทุกคน {everyone} มีเพื่อนใหม่เข้ามาอย่าลืมทักทายกันนะ!",
			"quickReply": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"type": "action",
						"action": map[string]interface{}{
							"type":  "uri",
							"label": "เริ่มทำแบบทดสอบ",
							"uri":   liffURL,
						},
					},
					map[string]interface{}{
						"type": "action",
						"action": map[string]interface{}{
							"type":  "message",
							"label": "Type",
							"text":  "Type",
						},
					},
				},
			},
			"substitution": map[string]interface{}{
				"user1": map[string]interface{}{
					"type": "mention",
					"mentionee": map[string]interface{}{
						"type":   "user",
						"userId": userID,
					},
				},
				"everyone": map[string]interface{}{
					"type": "mention",
					"mentionee": map[string]interface{}{
						"type": "all",
					},
				},
			},
		}
		utils.ReplyMessage(replyToken, []interface{}{message})
		log.Println("✅ Welcomed new member:", userID)
	}
}

func handleMessageEvent(event map[string]interface{}, w http.ResponseWriter) {
	message := event["message"].(map[string]interface{})
	text := message["text"].(string)
	replyToken := event["replyToken"].(string)
	source := event["source"].(map[string]interface{})
	groupID := source["groupId"].(string)
	userID := source["userId"].(string)

	if text == "ฉันได้ประเมินเรียบร้อยแล้ว" || text == "Type" {
		userData, err := utils.GetAnswersByUserID(userID, groupID)
		if err != nil {
			log.Println("❌ Failed to get user answers:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		liffURL := os.Getenv("LINE_LIFF_DISC") + "?groupId=" + groupID

		var response map[string]interface{}
		if userData != nil {
			response = map[string]interface{}{
				"type":       "textV2",
				"text":       fmt.Sprintf("คุณ {user1} คุณอยู่ในกลุ่ม %s \r\n\r\n รายละเอียด %s", userData["model"], userData["description"]),
				"quoteToken": message["quoteToken"],
				"quickReply": createQuickReplyItems(liffURL),
				"substitution": map[string]interface{}{
					"user1": map[string]interface{}{
						"type": "mention",
						"mentionee": map[string]interface{}{
							"type":   "user",
							"userId": userID,
						},
					},
				},
			}
		} else {
			response = map[string]interface{}{
				"type":       "textV2",
				"text":       "สวัสดีครับ {user1} เรามาเริ่มทำแบบทดสอบกันดีกว่า",
				"quoteToken": message["quoteToken"],
				"quickReply": createQuickReplyItems(liffURL),
				"substitution": map[string]interface{}{
					"user1": map[string]interface{}{
						"type": "mention",
						"mentionee": map[string]interface{}{
							"type":   "user",
							"userId": userID,
						},
					},
				},
			}
		}

		utils.ReplyMessage(replyToken, []interface{}{response})
	}

	if text == "วิเคราะห์" {
		userList, err := utils.GetAllUsersInGroup(groupID)
		fmt.Println("userList", userList)

		if err != nil {
			log.Println("❌ Failed to get users in group:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if len(userList) == 0 {
			log.Println("⚠️ No user data found for group:", groupID)
			utils.ReplyMessage(replyToken, []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "ไม่พบข้อมูลของผู้ใช้ในกลุ่มนี้ โปรดทำแบบทดสอบก่อนนะครับ 🙏",
				},
			})
			return
		}

		// ✅ เตรียมข้อความและแท็ก mention
		var contentBuilder strings.Builder
		substitution := map[string]interface{}{}
		count := map[string]int{"D": 0, "I": 0, "S": 0, "C": 0}

		fmt.Println("count")

		for idx, user := range userList {

			fmt.Println(user)

			userID := user["userId"].(string)
			model := user["model"].(string)
			mentionKey := fmt.Sprintf("user%d", idx)

			count[string(model[0])]++
			contentBuilder.WriteString(fmt.Sprintf("- {%s} อยู่ในกลุ่ม %s\n", mentionKey, model))

			substitution[mentionKey] = map[string]interface{}{
				"type": "mention",
				"mentionee": map[string]interface{}{
					"type":   "user",
					"userId": userID,
				},
			}
		}

		// ✅ สรุปและคำแนะนำ
		contentBuilder.WriteString("\n👥 สรุปจำนวน DISC:\n")
		contentBuilder.WriteString(fmt.Sprintf("D: %d | I: %d | S: %d | C: %d\n", count["D"], count["I"], count["S"], count["C"]))

		contentBuilder.WriteString("\n📌 แนะนำการจับคู่ DISC ที่ทำงานเข้ากันได้:\n")
		contentBuilder.WriteString("- D + I: เด็ดขาด + สื่อสารเก่ง\n")
		contentBuilder.WriteString("- D + C: ตัดสินใจไว + วิเคราะห์เก่ง\n")
		contentBuilder.WriteString("- I + S: บรรยากาศดี + ทีมเวิร์ค\n")
		contentBuilder.WriteString("- S + C: มั่นคง + ละเอียด\n")

		fmt.Println(contentBuilder)

		// ✅ ส่งข้อความ reply แบบ textV2 พร้อม mention
		message := map[string]interface{}{
			"type":         "textV2",
			"text":         contentBuilder.String(),
			"substitution": substitution,
			"quickReply": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"type": "action",
						"action": map[string]interface{}{
							"type":  "uri",
							"label": "ทำแบบสอบถาม",
							"uri":   os.Getenv("LINE_LIFF_DISC") + "?groupId=" + groupID,
						},
					},
				},
			},
		}

		fmt.Println(message)

		utils.ReplyMessage(replyToken, []interface{}{message})
	}

	mentionees, ok := event["message"].(map[string]interface{})["mention"].(map[string]interface{})["mentionees"].([]interface{})
	if ok {
		liffURL := os.Getenv("LINE_LIFF_DISC") + "?groupId=" + groupID
		for _, mentionee := range mentionees {
			mentioneeMap := mentionee.(map[string]interface{})

			fmt.Println("--------------------------------")
			fmt.Println(mentioneeMap)
			fmt.Println("--------------------------------")
			if isSelfVal, ok := mentioneeMap["isSelf"]; ok && isSelfVal != nil {
				if isSelf, ok := isSelfVal.(bool); ok && isSelf {
					// ทำงานต่อเมื่อ isSelf เป็น true
					if mentioneeMap["isSelf"].(bool) {
						response := map[string]interface{}{
							"type":       "textV2",
							"text":       "ว่ายังไงครับ ถามได้เลย {user1}",
							"quoteToken": event["message"].(map[string]interface{})["quoteToken"],
							"substitution": map[string]interface{}{
								"user1": map[string]interface{}{
									"type": "mention",
									"mentionee": map[string]interface{}{
										"type":   "user",
										"userId": userID,
									},
								},
							},
							"quickReply": map[string]interface{}{
								"items": []interface{}{
									map[string]interface{}{
										"type": "action",
										"action": map[string]interface{}{
											"type":  "uri",
											"label": "เริ่มทำแบบทดสอบ",
											"uri":   liffURL,
										},
									},
									map[string]interface{}{
										"type": "action",
										"action": map[string]interface{}{
											"type":  "message",
											"label": "Type",
											"text":  "Type",
										},
									},
								},
							},
						}
						utils.ReplyMessage(replyToken, []interface{}{response})
					}
				}
			}
			if mentioneeMap["type"].(string) == "all" {
				response := map[string]interface{}{
					"type":       "textV2",
					"text":       "ว่ายังไงครับ ถามได้เลย",
					"quoteToken": event["message"].(map[string]interface{})["quoteToken"],
					"quickReply": map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{
								"type": "action",
								"action": map[string]interface{}{
									"type":  "uri",
									"label": "เริ่มทำแบบทดสอบ",
									"uri":   liffURL,
								},
							},
							map[string]interface{}{
								"type": "action",
								"action": map[string]interface{}{
									"type":  "message",
									"label": "Type",
									"text":  "Type",
								},
							},
						},
					},
				}
				utils.ReplyMessage(replyToken, []interface{}{response})
			}
		}
	}

}

func createQuickReplyItems(liffURL string) map[string]interface{} {
	return map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"type": "action",
				"action": map[string]interface{}{
					"type":  "uri",
					"label": "ทำแบบทดสอบ",
					"uri":   liffURL,
				},
			},
			map[string]interface{}{
				"type": "action",
				"action": map[string]interface{}{
					"type":  "message",
					"label": "Type",
					"text":  "Type",
				},
			},
		},
	}
}

func handleLeaveEvent(event map[string]interface{}) {
	groupID := event["source"].(map[string]interface{})["groupId"].(string)
	log.Println("👋 Bot left group:", groupID)

	if err := utils.DeleteGroup(groupID); err != nil {
		log.Println("❌ Failed to delete group:", err)
		return
	}
	log.Println("✅ Group deleted successfully")
}

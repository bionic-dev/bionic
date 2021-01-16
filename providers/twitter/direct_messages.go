package twitter

import (
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io/ioutil"
	"strings"
	"time"
)

type Conversation struct {
	gorm.Model
	ID             string          `json:"conversationId"`
	DirectMessages []DirectMessage `json:"messages"`
}

func (Conversation) TableName() string {
	return tablePrefix + "conversations"
}

func (c *Conversation) UnmarshalJSON(b []byte) error {
	type alias Conversation

	var data struct {
		DmConversation alias `json:"dmConversation"`
	}

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	*c = Conversation(data.DmConversation)

	return nil
}

type DirectMessage struct {
	gorm.Model
	ConversationID string
	ID             int                     `json:"id,string"`
	RecipientID    int                     `json:"recipientId,string"`
	Reactions      []DirectMessageReaction `json:"reactions"`
	URLs           []URL                   `json:"urls" gorm:"many2many:twitter_direct_message_urls"`
	Text           string                  `json:"text"`
	MediaURLs      []string                `json:"mediaUrls" gorm:"type:text"`
	SenderID       int                     `json:"senderId,string"`
	Created        time.Time               `json:"createdAt"`
}

func (DirectMessage) TableName() string {
	return tablePrefix + "direct_messages"
}

func (dm *DirectMessage) UnmarshalJSON(b []byte) error {
	type alias DirectMessage

	var data struct {
		MessageCreate alias `json:"messageCreate"`
	}

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	*dm = DirectMessage(data.MessageCreate)

	return nil
}

type DirectMessageReaction struct {
	gorm.Model
	DirectMessageID int       `gorm:"uniqueIndex:twitter_direct_message_reactions_key"`
	SenderID        string    `json:"senderId"`
	Key             string    `json:"reactionKey" gorm:"uniqueIndex:twitter_direct_message_reactions_key"`
	EventID         string    `json:"eventId"`
	Created         time.Time `json:"createdAt"`
}

func (DirectMessageReaction) TableName() string {
	return tablePrefix + "direct_message_reactions"
}

func (p *twitter) importDirectMessages(inputPath string) error {
	var conversations []Conversation

	bytes, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return err
	}

	data := string(bytes)
	data = strings.TrimPrefix(data, "window.YTD.direct_messages.part0 = ")

	if err := json.Unmarshal([]byte(data), &conversations); err != nil {
		return err
	}

	for _, conversation := range conversations {
		for i, message := range conversation.DirectMessages {
			for j, reaction := range message.Reactions {
				err = p.DB().
					FirstOrCreate(&conversation.DirectMessages[i].Reactions[j], map[string]interface{}{
						"direct_message_id": message.ID,
						"key":               reaction.Key,
					}).
					Error
				if err != nil {
					return err
				}
			}

			for j, url := range message.URLs {
				err = p.DB().
					FirstOrCreate(&conversation.DirectMessages[i].URLs[j], map[string]interface{}{
						"url": url.URL,
					}).
					Error
				if err != nil {
					return err
				}
			}
		}
	}

	err = p.DB().
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		Create(conversations).
		Error
	if err != nil {
		return err
	}

	return nil
}

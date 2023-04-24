package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/kacperf531/sockchat/common"
	"github.com/mitchellh/mapstructure"
)

type MessageStore struct {
	es        *elasticsearch.Client
	indexName string
}

func NewMessageStore(es *elasticsearch.Client, indexName string) *MessageStore {
	return &MessageStore{es, indexName}
}

func getAllMessagesInChannelQuery(channel string) map[string]interface{} {
	return map[string]interface{}{
		"sort": []map[string]interface{}{{
			"timestamp": map[string]string{
				"order": "desc"}}},
		"query": map[string]interface{}{
			"match": map[string]string{
				"channel": channel,
			},
		},
	}
}

func searchMessagesInChannelQuery(channel, soughtPhrase string) map[string]interface{} {
	return map[string]interface{}{
		"sort": []map[string]interface{}{{
			"timestamp": map[string]string{
				"order": "desc"}}},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"channel": map[string]string{
								"query":    channel,
								"operator": "and",
							},
						},
					},
					{
						"match": map[string]interface{}{
							"text": map[string]string{
								"query": soughtPhrase,
							},
						},
					},
				},
			},
		},
	}
}

func (s *MessageStore) runSearchQuery(query map[string]interface{}) ([]*common.MessageEvent, error) {
	var (
		buf bytes.Buffer
		r   map[string]interface{}
	)
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.Fatalf("Error encoding query: %s", err)
	}

	res, err := s.es.Search(
		s.es.Search.WithContext(context.Background()),
		s.es.Search.WithIndex(s.indexName),
		s.es.Search.WithBody(&buf),
		s.es.Search.WithTrackTotalHits(true),
		s.es.Search.WithPretty(),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Fatalf("Error parsing the response body: %s", err)
		} else {
			log.Fatalf(
				"[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Fatalf("Error parsing the response body: %s", err)
	}
	var results []*common.MessageEvent
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		msg := &common.MessageEvent{}
		err := mapstructure.Decode(hit.(map[string]interface{})["_source"], &msg)
		if err == nil {
			results = append(results, msg)
		} else {
			log.Printf("Error decoding message: %v", err)
		}
	}

	return results, nil
}

func (s *MessageStore) GetMessagesByChannel(channel string) ([]*common.MessageEvent, error) {
	return s.runSearchQuery(getAllMessagesInChannelQuery(channel))
}

func (s *MessageStore) SearchMessagesInChannel(channel, phrase string) ([]*common.MessageEvent, error) {
	return s.runSearchQuery(searchMessagesInChannelQuery(channel, phrase))
}

func (s *MessageStore) IndexMessage(msg *common.MessageEvent) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("could not save message due to marshalling error")
	}

	req := esapi.IndexRequest{
		Index:        s.indexName,
		DocumentType: "message",
		Body:         bytes.NewReader(data),
		Refresh:      "true",
	}

	res, err := req.Do(context.Background(), s.es)
	if err != nil {
		return "", fmt.Errorf("could not save message due to DB error: %v", err)
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error while saving message - could not read DB response: %v", err)
	}
	var newDocument struct {
		Id string `json:"_id"`
	}
	if err := json.Unmarshal(bodyBytes, &newDocument); err == nil {
		return newDocument.Id, nil
	}
	return "", fmt.Errorf("could not unmarshal response from DB when saving message: %v", err)

}

package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/kacperf531/sockchat/common"
	"github.com/mitchellh/mapstructure"
)

const (
	sortByTimestamp = `"sort" : [ { "timestamp" : { "order" : "desc" } }]`
	filterByChannel = `"filter": [
		{
			"term": {
				"channel.keyword": {
					"value": %q
				}
			}
		}
	]`
	matchByPhrase = `"must": {
		"match_phrase_prefix": {
			"text": %q
		}
	}`
)

type MessageStore struct {
	es        *elasticsearch.Client
	indexName string
}

func NewMessageStore(es *elasticsearch.Client, indexName string) *MessageStore {
	return &MessageStore{es, indexName}
}

func (s *MessageStore) buildSearchQuery(channel, soughtPhrase string) io.Reader {
	var b strings.Builder

	b.WriteString("{\n")
	b.WriteString(`"query": {`)
	b.WriteString(`"bool": {`)
	b.WriteString(fmt.Sprintf(filterByChannel, channel))
	if soughtPhrase != "" {
		b.WriteString("," + fmt.Sprintf(matchByPhrase, soughtPhrase))

	}
	b.WriteString("}")
	b.WriteString("},")
	b.WriteString(sortByTimestamp)
	b.WriteString("\n}")

	return strings.NewReader(b.String())
}

func (s *MessageStore) runSearchQuery(query io.Reader) ([]*common.MessageEvent, error) {

	res, err := s.es.Search(
		s.es.Search.WithContext(context.Background()),
		s.es.Search.WithIndex(s.indexName),
		s.es.Search.WithBody(query),
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

	var r map[string]interface{}
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
	query := s.buildSearchQuery(channel, "")
	return s.runSearchQuery(query)
}

func (s *MessageStore) SearchMessagesInChannel(channel, phrase string) ([]*common.MessageEvent, error) {
	query := s.buildSearchQuery(channel, phrase)
	return s.runSearchQuery(query)
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

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

type searchQuery struct {
	Query struct {
		Bool boolQueryFilter `json:"bool"`
	} `json:"query"`
	Sort []timestampOrder `json:"sort"`
}

type boolQueryFilter struct {
	Filter []termFilter `json:"filter"`
	Must   *must        `json:"must,omitempty"`
}

type must struct {
	Match struct {
		Text struct {
			Query     string `json:"query"`
			Fuzziness string `json:"fuzziness"`
		} `json:"text"`
	} `json:"match"`
}

type termFilter struct {
	Term struct {
		Channel struct {
			Value string `json:"value"`
		} `json:"channel.keyword"`
	} `json:"term"`
}

type timestampOrder struct {
	Timestamp struct {
		Order string `json:"order"`
	} `json:"timestamp"`
}

func (s *MessageStore) buildSearchQuery(channel, soughtPhrase string) io.Reader {
	var q searchQuery

	if soughtPhrase != "" {
		m := &must{}
		m.Match.Text.Query = soughtPhrase
		m.Match.Text.Fuzziness = "AUTO"
		q.Query.Bool.Must = m
	}

	var tf termFilter
	tf.Term.Channel.Value = channel
	q.Query.Bool.Filter = []termFilter{tf}

	var ts timestampOrder
	ts.Timestamp.Order = "desc"
	q.Sort = []timestampOrder{ts}

	qJson, err := json.Marshal(&q)
	if err != nil {
		log.Fatalf("Error parsing the elastic search query: %s", err)
	}

	return bytes.NewReader(qJson)
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

func (s *MessageStore) FindMessages(channel, phrase string) ([]*common.MessageEvent, error) {
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

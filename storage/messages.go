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
	"github.com/kacperf531/sockchat/api"
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
	Sort []timestampOrder `json:"sort,omitempty"`
	Aggs interface{}      `json:"aggs,omitempty"`
}

type boolQueryFilter struct {
	Filter []filters `json:"filter"`
	Must   *must     `json:"must,omitempty"`
}

type must struct {
	Match struct {
		Text struct {
			Query     string `json:"query"`
			Fuzziness string `json:"fuzziness"`
		} `json:"text"`
	} `json:"match"`
}

type filters struct {
	Term  *term   `json:"term,omitempty"`
	Range *range_ `json:"range,omitempty"`
}

type term struct {
	Channel *termFilterValue `json:"channel.keyword,omitempty"`
	Author  *termFilterValue `json:"author.keyword,omitempty"`
}

type range_ struct {
	Timestamp struct {
		Gte int64 `json:"gte"`
		Lte int64 `json:"lte"`
	} `json:"timestamp"`
}

type termFilterValue struct {
	Value string `json:"value"`
}

type timestampOrder struct {
	Timestamp struct {
		Order string `json:"order"`
	} `json:"timestamp"`
}

func (s *MessageStore) buildSearchQuery(channel, soughtPhrase string) (*bytes.Reader, error) {
	var q searchQuery

	if soughtPhrase != "" {
		m := &must{}
		m.Match.Text.Query = soughtPhrase
		m.Match.Text.Fuzziness = "AUTO"
		q.Query.Bool.Must = m
	}

	filter := filters{
		Term: &term{Channel: &termFilterValue{Value: channel}},
	}
	q.Query.Bool.Filter = []filters{filter}

	var ts timestampOrder
	ts.Timestamp.Order = "desc"
	q.Sort = []timestampOrder{ts}

	qJson, err := json.Marshal(&q)
	if err != nil {
		log.Printf("Error marshalling query to es: %s", err)
		return nil, err
	}

	return bytes.NewReader(qJson), nil
}

func (s *MessageStore) runSearchQuery(ctx context.Context, query io.Reader) (api.ChannelHistory, error) {

	res, err := s.es.Search(
		s.es.Search.WithContext(ctx),
		s.es.Search.WithIndex(s.indexName),
		s.es.Search.WithBody(query),
		s.es.Search.WithTrackTotalHits(true),
		s.es.Search.WithPretty(),
	)
	if err != nil {
		log.Printf("Error getting response: %s", err)
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			log.Printf("error parsing the response body: %s", err)
			return nil, err
		} else {
			log.Printf("error returned from es: %s", e["error"].(map[string]interface{})["reason"])
			return nil, err
		}
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Printf("error parsing the es response: %s", err)
		return nil, err
	}
	var results api.ChannelHistory
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		msg := &api.MessageEvent{}
		err := mapstructure.Decode(hit.(map[string]interface{})["_source"], &msg)
		if err == nil {
			results = append(results, msg)
		} else {
			log.Printf("error decoding message from es: %s", err)
			return nil, err
		}
	}

	return results, nil
}

func (s *MessageStore) FindMessages(ctx context.Context, channel, phrase string) (api.ChannelHistory, error) {
	query, err := s.buildSearchQuery(channel, phrase)
	if err != nil {
		return nil, api.ErrInvalidRequest
	}
	results, err := s.runSearchQuery(ctx, query)
	if err != nil {
		return nil, api.ErrInternal
	}
	return results, nil
}
func (s *MessageStore) IndexMessage(msg *api.MessageEvent) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", api.ErrInvalidRequest
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

package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/kacperf531/sockchat/api"
)

const (
	dateFormat          = "yyyy-MM-dd HH:mm"
	dateFormatDays      = "yyyy-MM-dd"
	MaxChannelsInReport = 50
	MaxReportSizeInDays = 30
)

type UserReports struct {
	es        *elasticsearch.Client
	indexName string
}

func NewReportsService(es *elasticsearch.Client, indexName string) *UserReports {
	return &UserReports{es, indexName}
}

type UserActivityAggs struct {
	Channels struct {
		Terms struct {
			Field string `json:"field"`
			Size  int    `json:"size"`
		} `json:"terms"`
		Aggs *DateHistogramAggs `json:"aggs,omitempty"`
	} `json:"channels"`
}

type DateHistogramAggs struct {
	BySpecifiedRange struct {
		DateHistogram struct {
			Field            string `json:"field"`
			CalendarInterval string `json:"calendar_interval"`
			MinDocCount      int    `json:"min_doc_count"`
			Format           string `json:"format"`
			ExtendedBounds   struct {
				Min int64 `json:"min"`
				Max int64 `json:"max"`
			} `json:"extended_bounds"`
		} `json:"date_histogram"`
	} `json:"by_specified_range"`
}

type dateBucket struct {
	KeyString string `json:"key_as_string"`
	DocCount  int    `json:"doc_count"`
}

type channelBucket struct {
	Key              string      `json:"key"`
	DocCount         int         `json:"doc_count"`
	BySpecifiedRange *dateRanges `json:"by_specified_range,omitempty"`
}

type dateRanges struct {
	Buckets []dateBucket `json:"buckets"`
}

type channels struct {
	Buckets []channelBucket `json:"buckets"`
}
type aggs struct {
	Channels channels `json:"channels"`
}
type queryResult struct {
	Aggregations aggs `json:"aggregations"`
}

func (s *UserReports) runMessageAggregation(query io.Reader) (*queryResult, error) {
	res, err := s.es.Search(
		s.es.Search.WithContext(context.Background()),
		s.es.Search.WithIndex(s.indexName),
		s.es.Search.WithBody(query),
		s.es.Search.WithTrackTotalHits(true),
		s.es.Search.WithPretty(),
	)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		log.Printf("error running user activity es query: %v", res)
		return nil, api.ErrInternal
	}
	defer res.Body.Close()
	queryResult, err := unmarshalMessageAggregationQueryResult(res)
	if err != nil {
		return nil, err
	}
	return queryResult, nil
}

func (s *UserReports) GetUserActivityReport(opts *api.UserActivityReportOptions) (*api.UserActivityReport, error) {
	q, err := buildUserActivityQuery(opts)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	err = json.NewEncoder(&b).Encode(q)
	if err != nil {
		log.Printf("error encoding user activity query to es: %v", err)
		return nil, api.ErrInvalidRequest
	}

	queryResult, err := s.runMessageAggregation(&b)
	if err != nil {
		return nil, api.ErrInternal
	}

	results := newUserActivityReport(queryResult)
	results.From = opts.From
	results.To = opts.To
	return &results, nil
}

func newUserActivityReport(queryResult *queryResult) api.UserActivityReport {
	channelActivity := make(map[string]*api.ChannelActivity)
	for _, channelBucket := range queryResult.Aggregations.Channels.Buckets {
		messageCountDistribution := newMessageCountDistribution(channelBucket.BySpecifiedRange)
		activityItem := &api.ChannelActivity{TotalMessages: channelBucket.DocCount, MessageCountDistribution: messageCountDistribution}
		channelActivity[channelBucket.Key] = activityItem
	}
	return api.UserActivityReport{ChannelActivity: channelActivity}
}

func newMessageCountDistribution(res *dateRanges) []api.DistributionEntry {
	if res == nil {
		return nil
	}
	distribution := make([]api.DistributionEntry, 0, len(res.Buckets))
	for _, dateBucket := range res.Buckets {
		distribution = append(distribution, api.DistributionEntry{
			PeriodStart:      dateBucket.KeyString,
			MessagesInPeriod: dateBucket.DocCount,
		})
	}
	return distribution
}

func buildUserActivityQuery(opts *api.UserActivityReportOptions) (*searchQuery, error) {
	var tf term
	tf.Author = &termFilterValue{Value: opts.Author}

	var rf range_
	if opts.From.After(opts.To) {
		return nil, api.ErrInvalidRange
	}
	if opts.To.Sub(opts.From) > MaxReportSizeInDays*24*time.Hour {
		return nil, api.ErrMaxReportSizeExceeded
	}
	rf.Timestamp.Gte = opts.From.Unix()
	rf.Timestamp.Lte = opts.To.Unix()

	var aggs UserActivityAggs
	aggs.Channels.Terms.Field = "channel.keyword"
	aggs.Channels.Terms.Size = MaxChannelsInReport
	aggs.Channels.Aggs = buildDateHistogramAggs(opts)

	var q searchQuery
	q.Query.Bool.Filter = []filters{{Term: &tf}, {Range: &rf}}
	q.Aggs = aggs

	return &q, nil
}

func buildDateHistogramAggs(opts *api.UserActivityReportOptions) *DateHistogramAggs {
	if opts.GroupBy == "" {
		return nil
	}
	var dateSubaggs DateHistogramAggs
	dateSubaggs.BySpecifiedRange.DateHistogram.Field = "timestamp.as_date"
	dateSubaggs.BySpecifiedRange.DateHistogram.CalendarInterval = string(opts.GroupBy)
	dateSubaggs.BySpecifiedRange.DateHistogram.MinDocCount = 0

	if opts.GroupBy == api.GroupByDay {
		dateSubaggs.BySpecifiedRange.DateHistogram.Format = dateFormatDays
	} else {
		dateSubaggs.BySpecifiedRange.DateHistogram.Format = dateFormat
	}

	minTimestamp := opts.From.UnixMilli()
	maxTimestamp := opts.To.UnixMilli()
	dateSubaggs.BySpecifiedRange.DateHistogram.ExtendedBounds.Min = minTimestamp
	dateSubaggs.BySpecifiedRange.DateHistogram.ExtendedBounds.Max = maxTimestamp
	return &dateSubaggs
}

func unmarshalMessageAggregationQueryResult(res *esapi.Response) (*queryResult, error) {
	var r queryResult
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

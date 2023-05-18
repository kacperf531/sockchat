package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/kacperf531/sockchat/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIndexName   = "test_index"
	channelA        = "foo"
	channelB        = "bar"
	authorNick      = "FooBar420"
	testPeriodStart = "2020-01-01T00:00:00Z"
	testPeriodEnd   = "2020-01-02T23:59:59Z"
)

var testChannels = []string{channelA, channelB}

var (
	timeTestPeriodEnd, _        = time.Parse(time.RFC3339, testPeriodEnd)
	timestampWithinMinutePeriod = timeTestPeriodEnd.Unix() - 59
	timestampWithinHourPeriod   = timeTestPeriodEnd.Unix() - int64(time.Hour.Seconds()) + 1
	timestampWithinDayPeriod    = timeTestPeriodEnd.Unix() - int64(time.Hour.Seconds()*24) + 1
	testMessages                = []*api.MessageEvent{
		{
			Channel:   channelA,
			Text:      "text is irrelevant for these tests",
			Author:    authorNick,
			Timestamp: timestampWithinMinutePeriod,
		},
		{
			Channel:   channelB,
			Text:      "hello",
			Author:    authorNick,
			Timestamp: timestampWithinHourPeriod,
		},
		{
			Channel:   channelA,
			Text:      "some text",
			Author:    authorNick,
			Timestamp: timestampWithinDayPeriod,
		},
		{
			Channel:   channelB,
			Text:      "hello world",
			Author:    authorNick,
			Timestamp: timestampWithinDayPeriod,
		},
	}
)

var expectedMessageDistributions = map[string]map[int64]int{
	channelA: {
		timestampWithinMinutePeriod: 1,
		timestampWithinHourPeriod:   0,
		timestampWithinDayPeriod:    1,
	},
	channelB: {
		timestampWithinMinutePeriod: 0,
		timestampWithinHourPeriod:   1,
		timestampWithinDayPeriod:    1,
	},
}

func TestUserActivityReport(t *testing.T) {
	t.Parallel()

	es := mustSetUpES(t)
	setUpTestIndex(t, es, testIndexName)

	userReports := &UserReports{es, testIndexName}

	t.Run("returns report for user activity without group_by & to - records count is equal to number of channels", func(t *testing.T) {
		from, _ := time.Parse(time.RFC3339, testPeriodStart)
		to, _ := time.Parse(time.RFC3339, testPeriodEnd)
		report, err := userReports.GetUserActivityReport(&api.UserActivityReportOptions{Author: authorNick, From: from, To: to})
		require.NoError(t, err)
		require.Equal(t, 2, len((*report).ChannelActivity))
		assert.Equal(t, 2, (*report).ChannelActivity[channelA].TotalMessages)
		assert.Equal(t, 2, (*report).ChannelActivity[channelB].TotalMessages)
		assert.Equal(t, from, (*report).From)

	})

	t.Run("returns results grouped by day", func(t *testing.T) {
		dayOne := "2020-01-01"
		dayTwo := "2020-01-02"
		dayOneTime, _ := time.Parse(time.RFC3339, dayOne+"T00:00:00Z")
		dayTwoTime, _ := time.Parse(time.RFC3339, dayTwo+"T23:59:00Z")
		expectedEntriesDayReport := 2

		report, err := userReports.GetUserActivityReport(&api.UserActivityReportOptions{Author: authorNick, GroupBy: "day", From: dayOneTime, To: dayTwoTime})
		require.NoError(t, err)

		assert.Len(t, (*report).ChannelActivity[channelA].MessageCountDistribution, expectedEntriesDayReport)
		assert.Len(t, (*report).ChannelActivity[channelB].MessageCountDistribution, expectedEntriesDayReport)

		for _, tt := range testChannels {
			distributionDayOne := (*report).ChannelActivity[tt].MessageCountDistribution[0]
			distributionDayTwo := (*report).ChannelActivity[tt].MessageCountDistribution[1]

			expectedCountDayOne := 0
			expectedCountDayTwo := expectedMessageDistributions[tt][timestampWithinDayPeriod] + expectedMessageDistributions[tt][timestampWithinHourPeriod] + expectedMessageDistributions[tt][timestampWithinMinutePeriod]

			assert.Equal(t, expectedCountDayOne, distributionDayOne.MessagesInPeriod)
			assert.Equal(t, dayOne, distributionDayOne.PeriodStart)

			assert.Equal(t, expectedCountDayTwo, distributionDayTwo.MessagesInPeriod)
			assert.Equal(t, dayTwo, distributionDayTwo.PeriodStart)
		}
	})

	t.Run("returns results grouped by hour", func(t *testing.T) {
		from := time.Date(2020, 1, 2, 22, 0, 0, 0, time.UTC)
		to := time.Date(2020, 1, 2, 23, 59, 59, 0, time.UTC)
		expectedEntriesHourReport := 2

		report, err := userReports.GetUserActivityReport(&api.UserActivityReportOptions{Author: authorNick, GroupBy: "hour", From: from, To: to})
		require.NoError(t, err)

		require.Len(t, (*report).ChannelActivity[channelA].MessageCountDistribution, expectedEntriesHourReport)
		require.Len(t, (*report).ChannelActivity[channelB].MessageCountDistribution, expectedEntriesHourReport)

		for _, tt := range testChannels {
			distributionLastHour := (*report).ChannelActivity[tt].MessageCountDistribution[1]
			distributionPreviousHour := (*report).ChannelActivity[tt].MessageCountDistribution[0]

			expectedCountLastHour := expectedMessageDistributions[tt][timestampWithinMinutePeriod] + expectedMessageDistributions[tt][timestampWithinHourPeriod]
			expectedCountPreviousHour := 0

			assert.Equal(t, expectedCountLastHour, distributionLastHour.MessagesInPeriod)
			assert.Equal(t, "2020-01-02 23:00", distributionLastHour.PeriodStart)

			assert.Equal(t, expectedCountPreviousHour, distributionPreviousHour.MessagesInPeriod)
			assert.Equal(t, "2020-01-02 22:00", distributionPreviousHour.PeriodStart)
		}
	})

	t.Run("returns results grouped by minute", func(t *testing.T) {
		from := time.Date(2020, 1, 2, 23, 58, 0, 0, time.UTC)
		to := time.Date(2020, 1, 2, 23, 59, 59, 0, time.UTC)
		expectedEntriesMinuteReport := 2

		report, err := userReports.GetUserActivityReport(&api.UserActivityReportOptions{Author: authorNick, GroupBy: "minute", From: from, To: to})
		require.NoError(t, err)

		require.Len(t, (*report).ChannelActivity[channelA].MessageCountDistribution, expectedEntriesMinuteReport)

		distributionPreviousMinute := (*report).ChannelActivity[channelA].MessageCountDistribution[0]
		distributionLastMinute := (*report).ChannelActivity[channelA].MessageCountDistribution[1]

		expectedCountPreviousMinute := 0
		expectedCountLastMinute := expectedMessageDistributions[channelA][timestampWithinMinutePeriod]

		assert.Equal(t, expectedCountPreviousMinute, distributionPreviousMinute.MessagesInPeriod)
		assert.Equal(t, "2020-01-02 23:58", distributionPreviousMinute.PeriodStart)

		assert.Equal(t, expectedCountLastMinute, distributionLastMinute.MessagesInPeriod)
		assert.Equal(t, "2020-01-02 23:59", distributionLastMinute.PeriodStart)
	})

	t.Run("returns error when max report size is exceeded", func(t *testing.T) {
		from := time.Date(2023, 1, 2, 23, 58, 0, 0, time.UTC)
		to := time.Date(2023, 3, 2, 23, 59, 59, 0, time.UTC)

		_, err := userReports.GetUserActivityReport(&api.UserActivityReportOptions{Author: authorNick, GroupBy: "minute", From: from, To: to})
		require.ErrorIs(t, api.ErrMaxReportSizeExceeded, err)
	})

}

func setUpTestIndex(t *testing.T, es *elasticsearch.Client, indexName string) {
	t.Helper()
	_, err := es.Indices.Delete([]string{indexName})
	require.NoError(t, err)

	mapping, err := os.Open("timestamp_mapping.json")
	if err != nil {
		t.Error("could not load mapping file")
	}
	defer mapping.Close()

	require.NoError(t, err)
	indexReq := esapi.IndicesCreateRequest{
		Index: indexName,
	}
	_, err = indexReq.Do(context.Background(), es)
	require.NoError(t, err)

	messageStore := NewMessageStore(es, indexName)
	for _, msg := range testMessages {
		_, err := messageStore.IndexMessage(msg)
		require.NoError(t, err)
	}
	_, err = esapi.IndicesPutMappingRequest{Index: []string{indexName}, Body: mapping}.Do(context.Background(), es)
	require.NoError(t, err)
	_, err = esapi.UpdateByQueryRequest{Index: []string{indexName}}.Do(context.Background(), es)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // wait for ES reindex
}

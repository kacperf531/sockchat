package api

import "time"

const (
	GroupByDay        GroupBy = "day"
	GroupByHour       GroupBy = "hour"
	GroupByMinute     GroupBy = "minute"
	ReportsDateLayout string  = "2006-01-02 15:04"
)

// For messages sent from server
type MessageEvent struct {
	Text      string `json:"text"`
	Channel   string `json:"channel"`
	Author    string `json:"author"`
	Timestamp int64  `json:"timestamp"`
}

type PublicProfile struct {
	Nick        string `json:"nick"`
	Description string `json:"description"`
}

type ChannelHistory []*MessageEvent

type EmptyMessage struct{}

type GroupBy string

type UserActivityReportOptions struct {
	Author  string
	GroupBy GroupBy
	From    time.Time
	To      time.Time
}

type DistributionEntry struct {
	PeriodStart      string `json:"period_start"`
	MessagesInPeriod int    `json:"messages_in_period"`
}

type ChannelActivity struct {
	TotalMessages            int                 `json:"total_messages"`
	MessageCountDistribution []DistributionEntry `json:"message_count_distribution,omitempty"`
}

type UserActivityReport struct {
	ChannelActivity map[string]*ChannelActivity
	From            time.Time
	To              time.Time
}

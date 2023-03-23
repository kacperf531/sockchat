package sockchat

type CreateChannel struct {
	Name string `json:"name"`
}

type JoinChannel struct {
	Name string `json:"name"`
}

type ChannelJoined struct {
	ChannelName string `json:"channel"`
	UserName    string `json:"user"`
}

package sockchat

type CreateChannel struct {
	Name string `json:"name"`
}

type JoinChannel struct {
	Name string `json:"name"`
}

type ChannelJoined struct {
	Name     string `json:"name"`
	UserName string `json:"user"`
}

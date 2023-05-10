package api

type CreateProfileRequest struct {
	Nick        string `json:"nick"`
	Password    string `json:"password"`
	Description string `json:"description"`
}

type EditProfileRequest struct {
	Description string `json:"description"`
}

type GetProfileRequest struct {
	Nick string `json:"nick"`
}

type GetChannelHistoryRequest struct {
	Channel string `json:"channel"`
	Search  string `json:"search"`
}

type ErrorResponse struct {
	ErrorDescription string `json:"error_description"`
}

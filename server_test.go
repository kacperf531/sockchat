package sockchat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// StubChannelStore implements ChannelStore for testing purposes
type StubChannelStore struct {
	Channels map[string]int
}

func (*StubChannelStore) GetChannel(name string) int {
	panic("unimplemented")
}

func TestSockChat(t *testing.T) {

	t.Run("GET /channels returns 200", func(t *testing.T) {
		server := NewSockChatServer(&StubChannelStore{})

		request, _ := http.NewRequest(http.MethodGet, "/channels", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatusCode(t, response.Code, http.StatusOK)
	})

}

func assertStatusCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("got %d want %d", got, want)
	}
}

package roundtrip

import (
	"errors"
	"net/http"
	"testing"
)

var ErrNoMockResponse = errors.New("no mock response available")

type TestingRoundTripper struct {
	responses []*http.Response
	index     int

	t *testing.T
}

func (srt *TestingRoundTripper) WithTest(t *testing.T) *TestingRoundTripper {
	srt.t = t
	return srt
}

func (srt *TestingRoundTripper) WithMockResponses(responses []*http.Response) *TestingRoundTripper {
	srt.responses = responses
	return srt
}

func (srt *TestingRoundTripper) AddMockResponse(response *http.Response) *TestingRoundTripper {
	srt.responses = append(srt.responses, response)
	return srt
}

func (srt *TestingRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	if srt.index >= len(srt.responses) {
		if srt.t != nil {
			srt.t.Errorf("no mock response for request at index %d", srt.index)
		}
		return nil, ErrNoMockResponse
	}

	resp := srt.responses[srt.index]
	srt.index++

	return resp, nil
}

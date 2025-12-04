package roundtrip

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// This test the TestingRoundTripper to simulate the sequence of HTTP responses
// First a call with an expired token (401 Unauthorized)
// Then a call to refresh the token (200 OK with new token)
// Finally a call with the new token (200 OK with welcome message)
func TestTestingRoundTripper_RoundTrip(t *testing.T) {
	trt := &TestingRoundTripper{}
	trt.WithMockResponses([]*http.Response{
		newMockResponse(WithStatus(401), WithBody([]byte("token expired"))),
		newMockResponse(WithStatus(200), WithBody([]byte(`{"access_token":"newtok"}`))),
		newMockResponse(WithStatus(200), WithBody([]byte("welcome"))),
	})

	client := &http.Client{Transport: trt}

	// 1) initial call with expired JWT
	req1, _ := http.NewRequest("GET", "https://example.com/protected", nil)
	req1.Header.Set("Authorization", "Bearer expired")
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if resp1.StatusCode != 401 {
		t.Fatalf("expected 401 for expired token, got %d", resp1.StatusCode)
	}

	// 2) refresh token call to obtain a new access token
	req2, _ := http.NewRequest("POST", "https://example.com/refresh", nil)
	req2.Header.Set("Authorization", "Bearer refresh-token")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("refresh request failed: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200 from refresh endpoint, got %d", resp2.StatusCode)
	}
	b2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("reading refresh body: %v", err)
	}
	// naive extraction of token for test purposes
	newToken := ""
	if bytes.Contains(b2, []byte("access_token")) {
		newToken = "newtok"
	}

	// 3) call protected resource with new token
	req3, _ := http.NewRequest("GET", "https://example.com/protected", nil)
	req3.Header.Set("Authorization", "Bearer "+newToken)
	resp3, err := client.Do(req3)
	if err != nil {
		t.Fatalf("final request failed: %v", err)
	}
	if resp3.StatusCode != 200 {
		t.Fatalf("expected 200 after refresh, got %d", resp3.StatusCode)
	}
	b3, err := io.ReadAll(resp3.Body)
	if err != nil {
		t.Fatalf("reading final body: %v", err)
	}
	if string(b3) != "welcome" {
		t.Fatalf("expected 'welcome', got '%s'", string(b3))
	}
}

func TestTestingRoundTripper_AddMockResponse(t *testing.T) {
	t.Run("adds mock response with no previous MockResponses", func(t *testing.T) {
		trt := &TestingRoundTripper{}
		trt.AddMockResponse(newMockResponse(WithBody([]byte("response1"))))

		if len(trt.responses) != 1 {
			t.Errorf("expected 1 response, got %d", len(trt.responses))
		}

		b, err := io.ReadAll(trt.responses[0].Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		if string(b) != "response1" {
			t.Errorf("expected response 'response1', got '%s'", string(b))
		}
	})

	t.Run("adds mock response with previous MockResponses", func(t *testing.T) {
		trt := &TestingRoundTripper{
			responses: []*http.Response{newMockResponse(WithBody([]byte("response1")))},
		}
		trt.AddMockResponse(newMockResponse(WithBody([]byte("response2"))))

		if len(trt.responses) != 2 {
			t.Errorf("expected 2 responses, got %d", len(trt.responses))
		}
		b, err := io.ReadAll(trt.responses[1].Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		if string(b) != "response2" {
			t.Errorf("expected response 'response2', got '%s'", string(b))
		}
	})
}

func TestTestingRoundTripper_WithMockResponses(t *testing.T) {
	trt := &TestingRoundTripper{}
	trt.WithMockResponses([]*http.Response{
		newMockResponse(WithBody([]byte("response1"))),
		newMockResponse(WithBody([]byte("response2"))),
	})

	if len(trt.responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(trt.responses))
	}
	b, err := io.ReadAll(trt.responses[0].Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(b) != "response1" {
		t.Errorf("expected first response 'response1', got '%s'", string(b))
	}
	b2, err2 := io.ReadAll(trt.responses[1].Body)
	if err2 != nil {
		t.Fatalf("reading body: %v", err2)
	}
	if string(b2) != "response2" {
		t.Errorf("expected second response 'response2', got '%s'", string(b2))
	}
}

func TestTestingRoundTripper_WithTest(t *testing.T) {
	trt := &TestingRoundTripper{}
	trt.WithTest(t)

	if trt.t != t {
		t.Errorf("expected testing.T to be set")
	}

	// avoid failing the current test when RoundTrip calls t.Errorf for missing mocks
	trt.t = nil

	client := &http.Client{Transport: trt}
	_, err := client.Get("expected to fail due to no mock responses")
	if err == nil {
		t.Errorf("expected error due to no mock responses, got nil")
	}
}

func newMockResponse(opts ...func(*http.Response)) *http.Response {
	resp := &http.Response{
		StatusCode:    200,
		Status:        fmt.Sprintf("%d %s", 200, http.StatusText(200)),
		Body:          io.NopCloser(bytes.NewReader(nil)),
		Header:        make(http.Header),
		ContentLength: 0,
		Request:       nil,
	}
	for _, o := range opts {
		o(resp)
	}
	return resp
}

func WithStatus(status int) func(*http.Response) {
	return func(r *http.Response) {
		r.StatusCode = status
		r.Status = fmt.Sprintf("%d %s", status, http.StatusText(status))
	}
}

func WithBody(body []byte) func(*http.Response) {
	return func(r *http.Response) {
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	}
}

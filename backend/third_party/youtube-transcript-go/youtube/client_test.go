package youtube

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() should not return nil")
	}
	if client.httpClient == nil {
		t.Error("Client should have an httpClient")
	}
	if client.httpClient.Timeout != 45*time.Second {
		t.Errorf("Default timeout should be 45s, got %v", client.httpClient.Timeout)
	}
}

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if resp != "test response" {
		t.Errorf("Get() = %s, want 'test response'", resp)
	}
}

func TestClient_Get_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("Get() should return error for 404")
	}
	if _, ok := err.(*YouTubeRequestFailed); !ok {
		t.Errorf("Error should be YouTubeRequestFailed, got %T", err)
	}
}

func TestClient_Get_IpBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("Get() should return error for 429")
	}
	if _, ok := err.(*IpBlocked); !ok {
		t.Errorf("Error should be IpBlocked, got %T", err)
	}
}

func TestClient_Get_AcceptLanguageHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptLang := r.Header.Get("Accept-Language")
		if acceptLang != "en-US,en;q=0.9" {
			t.Errorf("Accept-Language header = %s, want 'en-US,en;q=0.9'", acceptLang)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type header = %s, want 'application/json'", contentType)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Post(server.URL, `{"test": "data"}`)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if !strings.Contains(resp, "success") {
		t.Errorf("Post() = %s, want JSON with 'success'", resp)
	}
}

func TestClient_Post_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Post(server.URL, `{}`)
	if err == nil {
		t.Fatal("Post() should return error for 500")
	}
	if _, ok := err.(*YouTubeRequestFailed); !ok {
		t.Errorf("Error should be YouTubeRequestFailed, got %T", err)
	}
}

func TestClient_Post_IpBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.Post(server.URL, `{}`)
	if err == nil {
		t.Fatal("Post() should return error for 429")
	}
	if _, ok := err.(*IpBlocked); !ok {
		t.Errorf("Error should be IpBlocked, got %T", err)
	}
}

func TestClient_GetWithCookies_Success(t *testing.T) {
	receivedCookie := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, cookie := range r.Cookies() {
			if cookie.Name == "test_cookie" && cookie.Value == "test_value" {
				receivedCookie = true
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient()
	cookies := []*http.Cookie{
		{Name: "test_cookie", Value: "test_value"},
	}
	_, err := client.GetWithCookies(server.URL, cookies)
	if err != nil {
		t.Fatalf("GetWithCookies() error = %v", err)
	}
	if !receivedCookie {
		t.Error("GetWithCookies() should send cookies")
	}
}

func TestClient_SetProxy(t *testing.T) {
	// Note: Testing actual proxy functionality is complex
	// This test just ensures SetProxy doesn't crash and properly sets the transport
	client := NewClient()
	err := client.SetProxy("http://proxy.example.com:8080")
	if err != nil {
		t.Fatalf("SetProxy() error = %v", err)
	}

	// Verify the transport was updated
	if _, ok := client.httpClient.Transport.(*http.Transport); !ok {
		t.Error("Transport should be *http.Transport after SetProxy")
	}
}

func TestClient_SetProxy_InvalidURL(t *testing.T) {
	client := NewClient()
	err := client.SetProxy("://invalid-url")
	if err == nil {
		t.Error("SetProxy() should return error for invalid URL")
	}
}

func TestClient_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Create a client with short timeout
	client := NewClient()
	client.httpClient.Timeout = 10 * time.Millisecond

	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("Get() should timeout with short timeout")
	}
}

func TestClient_Get_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if resp != "" {
		t.Errorf("Get() = '%s', want empty string", resp)
	}
}

func TestClient_Post_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Post(server.URL, "")
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	if resp != "ok" {
		t.Errorf("Post() = %s, want 'ok'", resp)
	}
}

func TestClient_Get_LargeResponse(t *testing.T) {
	largeBody := strings.Repeat("a", 1000000) // 1MB of 'a's
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(resp) != len(largeBody) {
		t.Errorf("Get() returned %d bytes, want %d", len(resp), len(largeBody))
	}
}

// Benchmark tests
func BenchmarkClient_Get(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Get(server.URL)
	}
}

func BenchmarkClient_Post(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Post(server.URL, `{}`)
	}
}

// Example usage test
func ExampleClient_Get() {
	client := NewClient()
	// In real usage, you would use a YouTube URL
	// This is just an example of the API
	_, err := client.Get("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Success")
}

func ExampleNewClient() {
	client := NewClient()
	fmt.Printf("Client created with timeout: %v\n", client.httpClient.Timeout)
}

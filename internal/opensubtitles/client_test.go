package opensubtitles

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSearch_SendsQueryLanguageAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "Inception" || r.URL.Query().Get("languages") != "zh-CN" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		if r.Header.Get("Api-Key") != "key-1" || r.Header.Get("User-Agent") != UserAgent {
			t.Fatalf("headers %+v", r.Header)
		}
		_, _ = w.Write([]byte(`{"data":[{"attributes":{"release":"Inception","language":"zh-CN","download_count":1,"ratings":9,"files":[{"file_id":7,"file_name":"inception.srt"}]}}]}`))
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	items, err := NewClient("u", "p", "key-1").Search(context.Background(), "Inception", "zh-CN")
	if err != nil || len(items) != 1 || items[0].FileID != 7 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestClientSearch_EmptyQueryError(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatalf("unexpected request %s", r.URL.Path)
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	_, err := NewClient("u", "p", "key-1").Search(context.Background(), "  ", "zh-CN")
	if err == nil || !strings.Contains(err.Error(), "搜索关键词不能为空") {
		t.Fatalf("err=%v", err)
	}
	if called {
		t.Fatal("empty query should fail before HTTP")
	}
}

func TestClientSearch_Non2xxUsesMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"quota exceeded"}`))
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	_, err := NewClient("u", "p", "key-1").Search(context.Background(), "Inception", "zh-CN")
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("err=%v", err)
	}
}

func TestClientSearch_Non2xxDefaultMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	_, err := NewClient("u", "p", "key-1").Search(context.Background(), "Inception", "zh-CN")
	if err == nil || err.Error() != "搜索字幕失败" {
		t.Fatalf("err=%v", err)
	}
}

func TestClientRequestDownload_RetriesLoginOn401(t *testing.T) {
	var logins, downloads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			logins++
			if r.Header.Get("Api-Key") != "key-1" || r.Header.Get("User-Agent") != UserAgent {
				t.Fatalf("login headers %+v", r.Header)
			}
			_, _ = w.Write([]byte(`{"token":"tok","status":200}`))
		case "/download":
			downloads++
			if downloads == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"invalid token"}`))
				return
			}
			_, _ = w.Write([]byte(`{"link":"http://example/file","file_name":"a.srt"}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	info, err := NewClient("u", "p", "key-1").RequestDownload(context.Background(), 7)
	if err != nil || downloads != 2 || logins < 1 || info.Link == "" {
		t.Fatalf("info=%+v err=%v logins=%d downloads=%d", info, err, logins, downloads)
	}
}

func TestClientRequestDownload_ReloginWhenCachedTokenRejected(t *testing.T) {
	var logins, downloads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			logins++
			if r.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("login content-type %s", r.Header.Get("Content-Type"))
			}
			body, _ := io.ReadAll(r.Body)
			var creds struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.Unmarshal(body, &creds); err != nil || creds.Username != "u" || creds.Password != "p" {
				t.Fatalf("login body=%s err=%v", body, err)
			}
			_, _ = w.Write([]byte(`{"token":"tok","status":200}`))
		case "/download":
			downloads++
			if r.Header.Get("Authorization") != "Bearer tok" {
				t.Fatalf("authorization %s", r.Header.Get("Authorization"))
			}
			if downloads == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"invalid token"}`))
				return
			}
			_, _ = w.Write([]byte(`{"link":"http://example/file","file_name":"a.srt"}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	info, err := NewClient("u", "p", "key-1").RequestDownload(context.Background(), 7)
	if err != nil || info.FileName != "a.srt" || info.Link == "" || downloads != 2 || logins < 1 {
		t.Fatalf("info=%+v err=%v logins=%d downloads=%d", info, err, logins, downloads)
	}
}

func TestClientRequestDownload_InvalidFileID(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	_, err := NewClient("u", "p", "key-1").RequestDownload(context.Background(), 0)
	if err == nil || !strings.Contains(err.Error(), "file_id 无效") {
		t.Fatalf("err=%v", err)
	}
	if called {
		t.Fatal("invalid file_id should fail before HTTP")
	}
}

func TestClientRequestDownload_LoginFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad credentials"}`))
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	_, err := NewClient("u", "p", "key-1").RequestDownload(context.Background(), 7)
	if !errors.Is(err, ErrLoginFailed) {
		t.Fatalf("err=%v", err)
	}
}

func TestClientFetchFile_GetsBodyAndUserAgent(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != UserAgent {
			t.Fatalf("ua %s", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte("srt-body"))
	}))
	defer server.Close()
	body, err := NewClient("u", "p", "key-1").FetchFile(context.Background(), server.URL)
	if err != nil || string(body) != "srt-body" {
		t.Fatalf("body=%q err=%v", body, err)
	}
}

func TestClientFetchFile_FollowsRedirect(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redir" {
			http.Redirect(w, r, "/file", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	body, err := NewClient("u", "p", "key-1").FetchFile(context.Background(), server.URL+"/redir")
	if err != nil || string(body) != "ok" {
		t.Fatalf("body=%q err=%v", body, err)
	}
}

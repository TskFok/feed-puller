package feishu

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBotService_SendText_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	svc := NewBotService(AppConfig{})
	err := svc.SendText(server.URL, "测试标题", "测试内容")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}

	var body botRequest
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.MsgType != "text" {
		t.Fatalf("msg_type = %q", body.MsgType)
	}
	if body.Content.Text != "测试标题\n\n测试内容" {
		t.Fatalf("text = %q", body.Content.Text)
	}
}

func TestBotService_SendText_EmptyWebhook(t *testing.T) {
	svc := NewBotService(AppConfig{})
	if err := svc.SendText("", "标题", "内容"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBotService_SendText_FeishuError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":9499,"msg":"invalid webhook url"}`))
	}))
	defer server.Close()

	svc := NewBotService(AppConfig{})
	err := svc.SendText(server.URL, "标题", "内容")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBotService_SendToUserByOpenID_EmptyConfig(t *testing.T) {
	svc := NewBotService(AppConfig{})
	if err := svc.SendToUserByOpenID("open_xxx", "标题", "内容"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBotService_SendViaAPI_EmptyConfig(t *testing.T) {
	svc := NewBotService(AppConfig{})
	if err := svc.SendViaAPI("", "secret", "open_id", "ou_xxx", "标题", "内容"); err == nil {
		t.Fatal("expected error")
	}
}

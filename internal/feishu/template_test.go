package feishu

import "testing"

func TestRenderTemplate(t *testing.T) {
	got := RenderTemplate("[完成] {{subscription}} / {{title}}", TemplateVars{
		Subscription: "动漫",
		Title:        "第1话",
	})
	if got != "[完成] 动漫 / 第1话" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplate_MediaType(t *testing.T) {
	got := RenderTemplate("[{{media_type}}] {{title}}", TemplateVars{MediaType: "电影", Title: "Demo"})
	if got != "[电影] Demo" {
		t.Fatalf("got %q", got)
	}
}

func TestParseRecipients(t *testing.T) {
	recipients := ParseRecipients("ou_legacy", "chat_id:oc_group\nou_user2")
	if len(recipients) != 3 {
		t.Fatalf("len = %d", len(recipients))
	}
	if recipients[0].ID != "ou_legacy" || recipients[1].IDType != "chat_id" {
		t.Fatalf("unexpected recipients: %+v", recipients)
	}
}

func TestParseRecipients_Dedupe(t *testing.T) {
	recipients := ParseRecipients("ou_same", "open_id:ou_same")
	if len(recipients) != 1 {
		t.Fatalf("len = %d", len(recipients))
	}
}

func TestCardPayload(t *testing.T) {
	payload := cardPayload(InteractiveCard{
		Title:    "测试",
		Template: "green",
		Lines:    []string{"**标题**: demo"},
	})
	header, ok := payload["header"].(map[string]any)
	if !ok {
		t.Fatal("missing header")
	}
	if header["template"] != "green" {
		t.Fatalf("template = %v", header["template"])
	}
}

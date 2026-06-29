package service

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestCreateModelProfileRedactsCredential(t *testing.T) {
	repo := newMemoryRepository()
	encryptor, err := NewCredentialEncryptor([]byte("12345678901234567890123456789012"), "local-v1")
	if err != nil {
		t.Fatalf("NewCredentialEncryptor() error = %v", err)
	}
	svc := New(repo, encryptor, 60000)

	profile, err := svc.CreateModelProfile(context.Background(), RequestContext{RequestID: "req-1", CallerService: "gateway"}, CreateModelProfileInput{
		Name:     "default-chat",
		Purpose:  PurposeChat,
		Provider: ProviderSiliconFlow,
		BaseURL:  "https://api.siliconflow.cn/v1",
		Model:    "Qwen/Qwen2.5",
		APIKey:   "sk-secret-value",
	})
	if err != nil {
		t.Fatalf("CreateModelProfile() error = %v", err)
	}
	if !profile.APIKeyConfigured {
		t.Fatalf("APIKeyConfigured = false, want true")
	}
	body, _ := json.Marshal(profile)
	if string(body) == "sk-secret-value" || bytes.Contains(body, []byte("sk-secret-value")) {
		t.Fatalf("profile response leaked api key: %s", body)
	}
	if got := repo.credentials[profile.CredentialID]; string(got.Ciphertext) == "sk-secret-value" || len(got.Nonce) == 0 {
		t.Fatalf("credential was not encrypted")
	}
}

func TestCreateModelProfileRejectsSensitiveDefaultParameters(t *testing.T) {
	svc := New(newMemoryRepository(), mustEncryptor(t), 60000)
	_, err := svc.CreateModelProfile(context.Background(), RequestContext{}, CreateModelProfileInput{
		Name:              "default-chat",
		Purpose:           PurposeChat,
		Provider:          ProviderSiliconFlow,
		BaseURL:           "https://api.siliconflow.cn/v1",
		Model:             "model",
		APIKey:            "sk-secret",
		DefaultParameters: json.RawMessage(`{"api_key":"nope"}`),
	})
	if err == nil {
		t.Fatalf("CreateModelProfile() error = nil, want validation error")
	}
}

func TestDefaultProfileConstraint(t *testing.T) {
	svc := New(newMemoryRepository(), mustEncryptor(t), 60000)
	isDefault := true

	cases := []struct {
		name       string
		purpose    Purpose
		dimensions *int
		topN       *int
	}{
		{name: "chat", purpose: PurposeChat},
		{name: "embedding", purpose: PurposeEmbedding, dimensions: intPtr(1024)},
		{name: "rerank", purpose: PurposeRerank, topN: intPtr(5)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := CreateModelProfileInput{
				Name:       "default-" + tc.name,
				Purpose:    tc.purpose,
				Provider:   ProviderSiliconFlow,
				BaseURL:    "https://api.siliconflow.cn/v1",
				Model:      "model",
				APIKey:     "sk-secret",
				IsDefault:  &isDefault,
				Dimensions: tc.dimensions,
				TopN:       tc.topN,
			}
			if _, err := svc.CreateModelProfile(context.Background(), RequestContext{}, input); err != nil {
				t.Fatalf("first CreateModelProfile() error = %v", err)
			}
			input.Name = "second-" + tc.name
			if _, err := svc.CreateModelProfile(context.Background(), RequestContext{}, input); err != nil {
				t.Fatalf("second CreateModelProfile() error = %v", err)
			}
			items, err := svc.ListModelProfiles(context.Background(), ListModelProfilesFilter{})
			if err != nil {
				t.Fatalf("ListModelProfiles() error = %v", err)
			}
			defaults := 0
			for _, item := range items {
				if item.Purpose == tc.purpose && item.Enabled && item.IsDefault {
					defaults++
				}
			}
			if defaults != 1 {
				t.Fatalf("enabled %s defaults = %d, want 1", tc.purpose, defaults)
			}
		})
	}
}

func TestUpdateModelProfileRejectsEmptyAPIKey(t *testing.T) {
	svc := New(newMemoryRepository(), mustEncryptor(t), 60000)
	profile, err := svc.CreateModelProfile(context.Background(), RequestContext{}, CreateModelProfileInput{
		Name:     "default-chat",
		Purpose:  PurposeChat,
		Provider: ProviderSiliconFlow,
		BaseURL:  "https://api.siliconflow.cn/v1",
		Model:    "model",
		APIKey:   "sk-secret",
	})
	if err != nil {
		t.Fatalf("CreateModelProfile() error = %v", err)
	}
	empty := " "
	_, err = svc.UpdateModelProfile(context.Background(), RequestContext{}, UpdateModelProfileInput{
		ID:     profile.ID,
		APIKey: &empty,
	})
	if err == nil {
		t.Fatalf("UpdateModelProfile() error = nil, want validation error")
	}
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeValidation || appErr.Fields["apiKey"] == "" {
		t.Fatalf("UpdateModelProfile() error = %#v, want apiKey validation error", err)
	}
}

func intPtr(value int) *int {
	return &value
}

func mustEncryptor(t *testing.T) *CredentialEncryptor {
	t.Helper()
	encryptor, err := NewCredentialEncryptor([]byte("12345678901234567890123456789012"), "local-v1")
	if err != nil {
		t.Fatalf("NewCredentialEncryptor() error = %v", err)
	}
	return encryptor
}

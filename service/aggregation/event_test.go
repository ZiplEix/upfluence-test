package aggregation

import (
	"encoding/json"
	"testing"
)

func TestStreamEvent_AsItem(t *testing.T) {
	tests := []struct {
		name         string
		jsonPayload  string
		expectedType string
		expectFound  bool
	}{
		{
			name:         "Tweet",
			jsonPayload:  `{"tweet":{"id":1,"content":"hello","favorites":10,"retweets":2,"timestamp":100}}`,
			expectedType: "tweet",
			expectFound:  true,
		},
		{
			name:         "YoutubeVideo",
			jsonPayload:  `{"youtube_video":{"id":2,"name":"video","views":100,"likes":20,"comments":5,"timestamp":200}}`,
			expectedType: "youtube_video",
			expectFound:  true,
		},
		{
			name:         "InstagramMedia",
			jsonPayload:  `{"instagram_media":{"id":3,"text":"photo","likes":30,"comments":6,"views":300,"timestamp":300}}`,
			expectedType: "instagram_media",
			expectFound:  true,
		},
		{
			name:         "FacebookStatus",
			jsonPayload:  `{"facebook_status":{"id":4,"status":"update","likes":40,"comments":8,"timestamp":400}}`,
			expectedType: "facebook_status",
			expectFound:  true,
		},
		{
			name:         "Pin",
			jsonPayload:  `{"pin":{"id":5,"repins":10,"likes":50,"comments":7,"timestamp":500}}`,
			expectedType: "pin",
			expectFound:  true,
		},
		{
			name:         "Article",
			jsonPayload:  `{"article":{"id":6,"title":"news","likes":60,"comments":9,"timestamp":600}}`,
			expectedType: "article",
			expectFound:  true,
		},
		{
			name:         "EmptyEvent",
			jsonPayload:  `{}`,
			expectedType: "",
			expectFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event StreamEvent
			err := json.Unmarshal([]byte(tt.jsonPayload), &event)
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}

			item, ok := event.AsItem()
			if ok != tt.expectFound {
				t.Fatalf("expected found=%v, got %v", tt.expectFound, ok)
			}

			if tt.expectFound {
				if item == nil {
					t.Fatal("expected item to not be nil")
				}
				if item.Platform() != tt.expectedType {
					t.Errorf("expected platform %s, got %s", tt.expectedType, item.Platform())
				}
			} else {
				if item != nil {
					t.Errorf("expected item to be nil, got %v", item)
				}
			}
		})
	}
}

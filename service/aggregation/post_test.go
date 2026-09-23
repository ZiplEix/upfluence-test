package aggregation

import "testing"

func TestPostMetrics(t *testing.T) {
	t.Run("Tweet", func(t *testing.T) {
		tweet := &Tweet{Favorites: 42, Retweets: 10}

		if favs, ok := tweet.Metric("favorites"); !ok || favs != 42 {
			t.Errorf("expected favorites=42, ok=true; got %d, %v", favs, ok)
		}
		if rts, ok := tweet.Metric("retweets"); !ok || rts != 10 {
			t.Errorf("expected retweets=10, ok=true; got %d, %v", rts, ok)
		}
		if val, ok := tweet.Metric("likes"); ok || val != 0 {
			t.Errorf("expected likes unsupported for tweet, got %d, %v", val, ok)
		}
	})

	t.Run("YoutubeVideo", func(t *testing.T) {
		video := &YoutubeVideo{Likes: 250, Comments: 45}

		if likes, ok := video.Metric("likes"); !ok || likes != 250 {
			t.Errorf("expected likes=250, ok=true; got %d, %v", likes, ok)
		}
		if comments, ok := video.Metric("comments"); !ok || comments != 45 {
			t.Errorf("expected comments=45, ok=true; got %d, %v", comments, ok)
		}
		if val, ok := video.Metric("favorites"); ok || val != 0 {
			t.Errorf("expected favorites unsupported, got %d, %v", val, ok)
		}
	})

	t.Run("InstagramMedia", func(t *testing.T) {
		insta := &InstagramMedia{Likes: 500, Comments: 60}

		if likes, ok := insta.Metric("likes"); !ok || likes != 500 {
			t.Errorf("expected likes=500, ok=true; got %d, %v", likes, ok)
		}
		if comments, ok := insta.Metric("comments"); !ok || comments != 60 {
			t.Errorf("expected comments=60, ok=true; got %d, %v", comments, ok)
		}
		if val, ok := insta.Metric("retweets"); ok || val != 0 {
			t.Errorf("expected retweets unsupported, got %d, %v", val, ok)
		}
	})

	t.Run("FacebookStatus", func(t *testing.T) {
		fb := &FacebookStatus{Likes: 120, Comments: 15}

		if likes, ok := fb.Metric("likes"); !ok || likes != 120 {
			t.Errorf("expected likes=120, ok=true; got %d, %v", likes, ok)
		}
		if comments, ok := fb.Metric("comments"); !ok || comments != 15 {
			t.Errorf("expected comments=15, ok=true; got %d, %v", comments, ok)
		}
		if val, ok := fb.Metric("shares"); ok || val != 0 {
			t.Errorf("expected shares unsupported, got %d, %v", val, ok)
		}
	})

	t.Run("Pin", func(t *testing.T) {
		pin := &Pin{Likes: 80, Comments: 8}

		if likes, ok := pin.Metric("likes"); !ok || likes != 80 {
			t.Errorf("expected likes=80, ok=true; got %d, %v", likes, ok)
		}
		if comments, ok := pin.Metric("comments"); !ok || comments != 8 {
			t.Errorf("expected comments=8, ok=true; got %d, %v", comments, ok)
		}
		if val, ok := pin.Metric("repins"); ok || val != 0 {
			t.Errorf("expected repins unsupported in Metric, got %d, %v", val, ok)
		}
	})

	t.Run("Article", func(t *testing.T) {
		article := &Article{Likes: 95, Comments: 22}

		if likes, ok := article.Metric("likes"); !ok || likes != 95 {
			t.Errorf("expected likes=95, ok=true; got %d, %v", likes, ok)
		}
		if comments, ok := article.Metric("comments"); !ok || comments != 22 {
			t.Errorf("expected comments=22, ok=true; got %d, %v", comments, ok)
		}
		if val, ok := article.Metric("unknown"); ok || val != 0 {
			t.Errorf("expected unknown unsupported, got %d, %v", val, ok)
		}
	})
}

package aggregation

// Item represent the unified contract of each stream item
type Item interface {
	// Platform return the origin platform of the item
	Platform() string

	// OccurredAt return a UNIX timestamp of the time the item was emited on the stream
	OccurredAt() int64

	// Metric returns the dimension's metric or false id the dimension is not available for this Item
	Metric(dimension string) (int, bool)
}

// ====================
// Platform specific implementation of the Item
// ====================

// Tweeter / X

type Tweet struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Favorites int    `json:"favorites"`
	Retweets  int    `json:"retweets"`
	Timestamp int64  `json:"timestamp"`
}

func (t *Tweet) Platform() string  { return "tweet" }
func (t *Tweet) OccurredAt() int64 { return t.Timestamp }
func (t *Tweet) Metric(dim string) (int, bool) {
	switch dim {
	case "favorites":
		return t.Favorites, true
	case "retweets":
		return t.Retweets, true
	default:
		return 0, false
	}
}

// Youtube

type YoutubeVideo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Views     int    `json:"views"`
	Likes     int    `json:"likes"`
	Comments  int    `json:"comments"`
	Timestamp int64  `json:"timestamp"`
}

func (y *YoutubeVideo) Platform() string  { return "youtube_video" }
func (y *YoutubeVideo) OccurredAt() int64 { return y.Timestamp }
func (y *YoutubeVideo) Metric(dim string) (int, bool) {
	switch dim {
	case "likes":
		return y.Likes, true
	case "comments":
		return y.Comments, true
	default:
		return 0, false
	}
}

// Instagram

type InstagramMedia struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	Likes     int    `json:"likes"`
	Comments  int    `json:"comments"`
	Views     int    `json:"views"`
	Timestamp int64  `json:"timestamp"`
}

func (i *InstagramMedia) Platform() string  { return "instagram_media" }
func (i *InstagramMedia) OccurredAt() int64 { return i.Timestamp }
func (i *InstagramMedia) Metric(dim string) (int, bool) {
	switch dim {
	case "likes":
		return i.Likes, true
	case "comments":
		return i.Comments, true
	default:
		return 0, false
	}
}

// Facebook

type FacebookStatus struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	Likes     int    `json:"likes"`
	Comments  int    `json:"comments"`
	Timestamp int64  `json:"timestamp"`
}

func (f *FacebookStatus) Platform() string  { return "facebook_status" }
func (f *FacebookStatus) OccurredAt() int64 { return f.Timestamp }
func (f *FacebookStatus) Metric(dim string) (int, bool) {
	switch dim {
	case "likes":
		return f.Likes, true
	case "comments":
		return f.Comments, true
	default:
		return 0, false
	}
}

// Pinterest

type Pin struct {
	ID        int64 `json:"id"`
	Repins    int   `json:"repins"`
	Likes     int   `json:"likes"`
	Comments  int   `json:"comments"`
	Timestamp int64 `json:"timestamp"`
}

func (p *Pin) Platform() string  { return "pin" }
func (p *Pin) OccurredAt() int64 { return p.Timestamp }
func (p *Pin) Metric(dim string) (int, bool) {
	switch dim {
	case "likes":
		return p.Likes, true
	case "comments":
		return p.Comments, true
	default:
		return 0, false
	}
}

// Article

type Article struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Likes     int    `json:"likes"`
	Comments  int    `json:"comments"`
	Timestamp int64  `json:"timestamp"`
}

func (a *Article) Platform() string  { return "article" }
func (a *Article) OccurredAt() int64 { return a.Timestamp }
func (a *Article) Metric(dim string) (int, bool) {
	switch dim {
	case "likes":
		return a.Likes, true
	case "comments":
		return a.Comments, true
	default:
		return 0, false
	}
}

package aggregation

// StreamEvent represent the structured JSON send by the Upfluence stream
type StreamEvent struct {
	Tweet          *Tweet          `json:"tweet,omitempty"`
	YoutubeVideo   *YoutubeVideo   `json:"youtube_video,omitempty"`
	InstagramMedia *InstagramMedia `json:"instagram_media,omitempty"`
	FacebookStatus *FacebookStatus `json:"facebook_status,omitempty"`
	Pin            *Pin            `json:"pin,omitempty"`
	Article        *Article        `json:"article,omitempty"`
}

// AsItem extract the entity as a Item interface
func (e *StreamEvent) AsItem() (Item, bool) {
	switch {
	case e.Tweet != nil:
		return e.Tweet, true
	case e.YoutubeVideo != nil:
		return e.YoutubeVideo, true
	case e.InstagramMedia != nil:
		return e.InstagramMedia, true
	case e.FacebookStatus != nil:
		return e.FacebookStatus, true
	case e.Pin != nil:
		return e.Pin, true
	case e.Article != nil:
		return e.Article, true
	default:
		return nil, false
	}
}

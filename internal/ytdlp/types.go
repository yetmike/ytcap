package ytdlp

type SearchResult struct {
	Type         string // "channel" or "video"
	ID           string
	Title        string
	URL          string
	Channel      string
	ChannelURL   string
	Duration     float64
	DurationStr  string
	ViewCount    int64
	Subscribers  string
	UploadDate   string
	Description  string
}

type Video struct {
	ID             string
	Title          string
	URL            string
	Channel        string
	ChannelURL     string
	Duration       float64
	DurationStr    string
	ViewCount      int64
	UploadDate     string
	Description    string
	Thumbnail      string
	PlaylistIndex  int // original position from yt-dlp (newest first)
}

type Channel struct {
	ID          string
	Title       string
	URL         string
	Subscribers string
	Description string
}

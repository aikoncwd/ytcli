// Package youtube wraps the yt-dlp binary for search, playlist expansion and
// metadata extraction. It never downloads media; mpv streams that directly.
package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/AikonCWD/ytcli/internal/track"
)

type Client struct{ bin string }

func New(ytDlpPath string) *Client { return &Client{bin: ytDlpPath} }

func buildSearchArgs(query string, n int) []string {
	return []string{fmt.Sprintf("ytsearch%d:%s", n, query), "-J", "--flat-playlist", "--no-warnings"}
}

func buildResolveArgs(url string) []string {
	return []string{url, "-J", "--flat-playlist", "--no-warnings"}
}

type rawEntry struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Channel    string   `json:"channel"`
	Uploader   string   `json:"uploader"`
	Duration   *float64 `json:"duration"`
	WebpageURL string   `json:"webpage_url"`
	URL        string   `json:"url"`
	IsLive     *bool    `json:"is_live"`
	LiveStatus string   `json:"live_status"`
}

type rawDump struct {
	rawEntry
	Entries *[]rawEntry `json:"entries"`
}

func (e rawEntry) toTrack() track.Track {
	ch := e.Channel
	if ch == "" {
		ch = e.Uploader
	}
	u := e.WebpageURL
	if u == "" {
		u = e.URL
	}
	if u == "" && e.ID != "" {
		u = "https://www.youtube.com/watch?v=" + e.ID
	}
	d := 0
	if e.Duration != nil {
		d = int(*e.Duration)
	}
	live := e.LiveStatus == "is_live" || (e.IsLive != nil && *e.IsLive)
	return track.Track{ID: e.ID, URL: u, Title: e.Title, Channel: ch, Duration: d, Live: live}
}

// parseDump reads yt-dlp -J output: a single video object, or a playlist
// object with an "entries" array (also used by ytsearch results).
func parseDump(b []byte) ([]track.Track, error) {
	var d rawDump
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	if d.Entries != nil {
		out := make([]track.Track, 0, len(*d.Entries))
		for _, e := range *d.Entries {
			out = append(out, e.toTrack())
		}
		return out, nil
	}
	return []track.Track{d.rawEntry.toTrack()}, nil
}

func (c *Client) run(args []string) ([]track.Track, error) {
	var out, stderr bytes.Buffer
	cmd := exec.Command(c.bin, args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp falló: %w: %s", err, stderr.String())
	}
	return parseDump(out.Bytes())
}

func (c *Client) Search(query string, n int) ([]track.Track, error) {
	return c.run(buildSearchArgs(query, n))
}

func (c *Client) Resolve(url string) ([]track.Track, error) {
	return c.run(buildResolveArgs(url))
}

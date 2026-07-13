// Package track defines the shared media item used across ytcli.
package track

// Track is a single playable YouTube item.
type Track struct {
	ID       string // YouTube video id
	URL      string // watch URL handed to mpv
	Title    string
	Channel  string
	Duration int // seconds; 0 if unknown
}

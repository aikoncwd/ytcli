// Package queue holds the in-memory playback queue: order, current position,
// shuffle and repeat. Pure logic with no external dependencies.
package queue

import (
	"math/rand"

	"github.com/AikonCWD/ytcli/internal/track"
)

// RepeatMode controls what Next/Prev do at the ends of the queue.
type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatAll
	RepeatOne
)

// Queue is the ordered list of tracks plus playback state.
type Queue struct {
	tracks  []track.Track
	order   []int // indices into tracks, in playback order
	pos     int   // index into order; -1 when empty
	shuffle bool
	repeat  RepeatMode
}

func New() *Queue { return &Queue{pos: -1, repeat: RepeatOff} }

func (q *Queue) Add(tracks ...track.Track) {
	start := len(q.tracks)
	q.tracks = append(q.tracks, tracks...)
	for i := range tracks {
		q.order = append(q.order, start+i)
	}
	if q.pos == -1 && len(q.order) > 0 {
		q.pos = 0
	}
}

func (q *Queue) Len() int          { return len(q.tracks) }
func (q *Queue) CurrentIndex() int { return q.pos }

func (q *Queue) Current() (track.Track, bool) {
	if q.pos < 0 || q.pos >= len(q.order) {
		return track.Track{}, false
	}
	return q.tracks[q.order[q.pos]], true
}

func (q *Queue) Tracks() []track.Track {
	out := make([]track.Track, 0, len(q.order))
	for _, i := range q.order {
		out = append(out, q.tracks[i])
	}
	return out
}

func (q *Queue) Next() (track.Track, bool) {
	if len(q.order) == 0 {
		return track.Track{}, false
	}
	switch q.repeat {
	case RepeatOne:
		// stay on current
	case RepeatAll:
		q.pos = (q.pos + 1) % len(q.order)
	default: // RepeatOff
		if q.pos+1 >= len(q.order) {
			return track.Track{}, false
		}
		q.pos++
	}
	return q.Current()
}

func (q *Queue) Prev() (track.Track, bool) {
	if len(q.order) == 0 {
		return track.Track{}, false
	}
	switch q.repeat {
	case RepeatOne:
		// stay
	case RepeatAll:
		q.pos = (q.pos - 1 + len(q.order)) % len(q.order)
	default:
		if q.pos-1 < 0 {
			return track.Track{}, false
		}
		q.pos--
	}
	return q.Current()
}

func (q *Queue) JumpTo(index int) (track.Track, bool) {
	if index < 0 || index >= len(q.order) {
		return track.Track{}, false
	}
	q.pos = index
	return q.Current()
}

func (q *Queue) SetShuffle(on bool) {
	if on == q.shuffle {
		return
	}
	q.shuffle = on
	cur := -1
	if q.pos >= 0 && q.pos < len(q.order) {
		cur = q.order[q.pos]
	}
	if on {
		rand.Shuffle(len(q.order), func(i, j int) {
			q.order[i], q.order[j] = q.order[j], q.order[i]
		})
	} else {
		for i := range q.order {
			q.order[i] = i
		}
	}
	if cur >= 0 {
		for i, idx := range q.order {
			if idx == cur {
				q.pos = i
				break
			}
		}
	}
}

func (q *Queue) Shuffle() bool             { return q.shuffle }
func (q *Queue) SetRepeat(m RepeatMode)    { q.repeat = m }
func (q *Queue) Repeat() RepeatMode        { return q.repeat }

package queue

import (
	"testing"

	"github.com/AikonCWD/ytcli/internal/track"
)

func tk(id string) track.Track { return track.Track{ID: id, URL: "u" + id} }

func ids(q *Queue) []string {
	var out []string
	for _, t := range q.Tracks() {
		out = append(out, t.ID)
	}
	return out
}

func TestRemoveAtBeforeCurrent(t *testing.T) {
	q := New()
	q.Add(tk("a"), tk("b"), tk("c"))
	q.JumpTo(2)
	if !q.RemoveAt(0) {
		t.Fatal("remove should succeed")
	}
	if cur, _ := q.Current(); cur.ID != "c" {
		t.Fatalf("current = %q; want c", cur.ID)
	}
	got := ids(q)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("tracks = %v; want [b c]", got)
	}
}

func TestRemoveAtCurrentMovesToNext(t *testing.T) {
	q := New()
	q.Add(tk("a"), tk("b"), tk("c"))
	q.JumpTo(1)
	q.RemoveAt(1)
	if cur, _ := q.Current(); cur.ID != "c" {
		t.Fatalf("current = %q; want c", cur.ID)
	}
}

func TestRemoveLastClampsPosition(t *testing.T) {
	q := New()
	q.Add(tk("a"), tk("b"))
	q.JumpTo(1)
	q.RemoveAt(1)
	if cur, _ := q.Current(); cur.ID != "a" {
		t.Fatalf("current = %q; want a", cur.ID)
	}
}

func TestRemoveOnlyTrackEmptiesQueue(t *testing.T) {
	q := New()
	q.Add(tk("a"))
	q.RemoveAt(0)
	if _, ok := q.Current(); ok || q.Len() != 0 {
		t.Fatalf("queue should be empty; len=%d", q.Len())
	}
	if q.RemoveAt(0) {
		t.Fatal("remove on empty queue should fail")
	}
}

func TestOriginalTracksIgnoresShuffle(t *testing.T) {
	q := New()
	q.Add(tk("a"), tk("b"), tk("c"), tk("d"), tk("e"))
	q.SetShuffle(true)
	got := q.OriginalTracks()
	for i, want := range []string{"a", "b", "c", "d", "e"} {
		if got[i].ID != want {
			t.Fatalf("OriginalTracks[%d] = %q; want %q (insertion order)", i, got[i].ID, want)
		}
	}
}

func TestIndexOfID(t *testing.T) {
	q := New()
	q.Add(tk("a"), tk("b"))
	if got := q.IndexOfID("b"); got != 1 {
		t.Fatalf("IndexOfID(b) = %d; want 1", got)
	}
	if got := q.IndexOfID("zz"); got != -1 {
		t.Fatalf("IndexOfID(zz) = %d; want -1", got)
	}
}

func TestRemoveAtWithShuffleKeepsMapping(t *testing.T) {
	q := New()
	q.Add(tk("a"), tk("b"), tk("c"), tk("d"))
	q.SetShuffle(true)
	before := ids(q)
	q.RemoveAt(2)
	after := ids(q)
	want := append(append([]string{}, before[:2]...), before[3:]...)
	if len(after) != 3 {
		t.Fatalf("len = %d; want 3", len(after))
	}
	for i := range want {
		if after[i] != want[i] {
			t.Fatalf("after = %v; want %v", after, want)
		}
	}
}

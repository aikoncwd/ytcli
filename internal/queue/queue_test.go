package queue

import (
	"sort"
	"testing"

	"github.com/AikonCWD/ytcli/internal/track"
)

func tracks(ids ...string) []track.Track {
	out := make([]track.Track, len(ids))
	for i, id := range ids {
		out[i] = track.Track{ID: id, URL: "u" + id, Title: id}
	}
	return out
}

func TestAddAndCurrent(t *testing.T) {
	q := New()
	if _, ok := q.Current(); ok {
		t.Fatal("empty queue should have no current")
	}
	q.Add(tracks("a", "b")...)
	cur, ok := q.Current()
	if !ok || cur.ID != "a" {
		t.Fatalf("current = %v, %v; want a", cur.ID, ok)
	}
	if q.Len() != 2 {
		t.Fatalf("len = %d; want 2", q.Len())
	}
}

func TestNextRepeatOff(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	if n, ok := q.Next(); !ok || n.ID != "b" {
		t.Fatalf("next = %v,%v; want b,true", n.ID, ok)
	}
	if _, ok := q.Next(); ok {
		t.Fatal("next past end should be false with RepeatOff")
	}
}

func TestNextRepeatAllWraps(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	q.SetRepeat(RepeatAll)
	q.Next()                    // b
	n, ok := q.Next()           // wrap to a
	if !ok || n.ID != "a" {
		t.Fatalf("wrap = %v,%v; want a,true", n.ID, ok)
	}
}

func TestNextRepeatOneStays(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	q.SetRepeat(RepeatOne)
	n, ok := q.Next()
	if !ok || n.ID != "a" {
		t.Fatalf("repeat-one next = %v,%v; want a,true", n.ID, ok)
	}
}

func TestPrevRepeatOff(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	if _, ok := q.Prev(); ok {
		t.Fatal("prev before start should be false")
	}
	q.Next() // b
	p, ok := q.Prev()
	if !ok || p.ID != "a" {
		t.Fatalf("prev = %v,%v; want a,true", p.ID, ok)
	}
}

func TestJumpTo(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b", "c")...)
	j, ok := q.JumpTo(2)
	if !ok || j.ID != "c" {
		t.Fatalf("jump = %v,%v; want c,true", j.ID, ok)
	}
	if _, ok := q.JumpTo(9); ok {
		t.Fatal("jump out of range should be false")
	}
}

func TestShufflePreservesSetAndCurrent(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b", "c", "d")...)
	q.Next() // current = b
	q.SetShuffle(true)
	cur, _ := q.Current()
	if cur.ID != "b" {
		t.Fatalf("shuffle changed current to %v; want b", cur.ID)
	}
	var ids []string
	for _, tr := range q.Tracks() {
		ids = append(ids, tr.ID)
	}
	sort.Strings(ids)
	want := []string{"a", "b", "c", "d"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("shuffle lost tracks: got %v", ids)
		}
	}
}

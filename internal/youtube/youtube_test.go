package youtube

import "testing"

func TestBuildSearchArgs(t *testing.T) {
	got := buildSearchArgs("daft punk", 15)
	want := []string{"ytsearch15:daft punk", "-J", "--flat-playlist", "--no-warnings"}
	if len(got) != len(want) {
		t.Fatalf("args = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestBuildResolveArgs(t *testing.T) {
	got := buildResolveArgs("https://youtu.be/x")
	want := []string{"https://youtu.be/x", "-J", "--flat-playlist", "--no-warnings"}
	if len(got) != len(want) {
		t.Fatalf("resolve args = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resolve args[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestParseDumpSingleVideo(t *testing.T) {
	js := []byte(`{"id":"abc","title":"Song","channel":"Chan","duration":211.0,"webpage_url":"https://youtu.be/abc"}`)
	got, err := parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d; want 1", len(got))
	}
	tr := got[0]
	if tr.ID != "abc" || tr.Title != "Song" || tr.Channel != "Chan" || tr.Duration != 211 || tr.URL != "https://youtu.be/abc" {
		t.Fatalf("track = %+v", tr)
	}
}

func TestParseDumpPlaylistUsesUploaderAndBuildsURL(t *testing.T) {
	js := []byte(`{"_type":"playlist","entries":[
		{"id":"1","title":"A","uploader":"U1"},
		{"id":"2","title":"B","channel":"C2","duration":10.0}
	]}`)
	got, err := parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d; want 2", len(got))
	}
	if got[0].Channel != "U1" {
		t.Fatalf("entry0 channel = %q; want U1 (uploader fallback)", got[0].Channel)
	}
	if got[0].URL != "https://www.youtube.com/watch?v=1" {
		t.Fatalf("entry0 url = %q; want built URL", got[0].URL)
	}
	if got[1].Duration != 10 {
		t.Fatalf("entry1 duration = %d; want 10", got[1].Duration)
	}
}

func TestParseDumpEmptyPlaylistReturnsNoTracks(t *testing.T) {
	js := []byte(`{"_type":"playlist","id":"PL123","title":"Empty","entries":[]}`)
	got, err := parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty playlist should yield 0 tracks, got %d: %+v", len(got), got)
	}
}

func TestParseDumpDetectsLiveStreams(t *testing.T) {
	// Single video resolve exposes is_live; flat-playlist search only live_status.
	js := []byte(`{"id":"live1","title":"Radio","is_live":true,"live_status":"is_live","duration":null,"webpage_url":"https://youtu.be/live1"}`)
	got, err := parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0].Live || got[0].Duration != 0 {
		t.Fatalf("track = %+v; want Live=true Duration=0", got[0])
	}

	js = []byte(`{"_type":"playlist","entries":[
		{"id":"a","title":"VOD","live_status":"was_live","duration":100.0},
		{"id":"b","title":"Radio","live_status":"is_live"}
	]}`)
	got, err = parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Live {
		t.Fatalf("was_live must not be Live: %+v", got[0])
	}
	if !got[1].Live {
		t.Fatalf("is_live entry should be Live: %+v", got[1])
	}
}

func TestParseDumpURLFallbackToURLField(t *testing.T) {
	js := []byte(`{"id":"z","title":"T","url":"https://example.com/z"}`)
	got, err := parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].URL != "https://example.com/z" {
		t.Fatalf("url fallback = %q; want the url field", got[0].URL)
	}
}

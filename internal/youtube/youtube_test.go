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
	if got[0] != "https://youtu.be/x" || got[1] != "-J" {
		t.Fatalf("resolve args = %v", got)
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

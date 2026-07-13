package player

import (
	"strings"
	"testing"
)

func TestCmdBuilders(t *testing.T) {
	cases := []struct {
		got  []byte
		want string
	}{
		{cmdLoad("http://x"), `{"command":["loadfile","http://x","replace"]}`},
		{cmdSetPause(true), `{"command":["set_property","pause",true]}`},
		{cmdSetPause(false), `{"command":["set_property","pause",false]}`},
		{cmdSeek(-10), `{"command":["seek",-10,"relative"]}`},
		{cmdSetVolume(80), `{"command":["set_property","volume",80]}`},
		{cmdObserve(1, "time-pos"), `{"command":["observe_property",1,"time-pos"]}`},
	}
	for _, c := range cases {
		got := strings.TrimRight(string(c.got), "\n")
		if got != c.want {
			t.Fatalf("cmd = %s; want %s", got, c.want)
		}
		if !strings.HasSuffix(string(c.got), "\n") {
			t.Fatalf("command must end with newline: %q", string(c.got))
		}
	}
}

func TestApplyLineProperties(t *testing.T) {
	var st State
	applyLine([]byte(`{"event":"property-change","name":"time-pos","data":42.7}`), &st)
	if st.Position != 42 {
		t.Fatalf("position = %d; want 42", st.Position)
	}
	applyLine([]byte(`{"event":"property-change","name":"duration","data":211.0}`), &st)
	applyLine([]byte(`{"event":"property-change","name":"volume","data":75.0}`), &st)
	applyLine([]byte(`{"event":"property-change","name":"pause","data":true}`), &st)
	applyLine([]byte(`{"event":"property-change","name":"media-title","data":"Song"}`), &st)
	if st.Duration != 211 || st.Volume != 75 || !st.Paused || st.Title != "Song" {
		t.Fatalf("state = %+v", st)
	}
}

func TestApplyLineReturnsEvent(t *testing.T) {
	var st State
	ev, reason, err := applyLine([]byte(`{"event":"end-file","reason":"eof"}`), &st)
	if err != nil || ev != "end-file" || reason != "eof" {
		t.Fatalf("got ev=%q reason=%q err=%v; want end-file,eof,nil", ev, reason, err)
	}
	ev, reason, _ = applyLine([]byte(`{"event":"end-file","reason":"stop"}`), &st)
	if ev != "end-file" || reason != "stop" {
		t.Fatalf("stop event: got ev=%q reason=%q; want end-file,stop", ev, reason)
	}
}

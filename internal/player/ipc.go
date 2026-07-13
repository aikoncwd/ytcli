package player

import "encoding/json"

func cmdJSON(parts ...interface{}) []byte {
	b, _ := json.Marshal(map[string]interface{}{"command": parts})
	return append(b, '\n')
}

func cmdLoad(url string) []byte { return cmdJSON("loadfile", url, "replace") }
func cmdSetPause(p bool) []byte { return cmdJSON("set_property", "pause", p) }
func cmdSeek(delta int) []byte  { return cmdJSON("seek", delta, "relative") }
func cmdSetVolume(v int) []byte { return cmdJSON("set_property", "volume", v) }
func cmdObserve(id int, name string) []byte {
	return cmdJSON("observe_property", id, name)
}

type eventMsg struct {
	Event string          `json:"event"`
	Name  string          `json:"name"`
	Data  json.RawMessage `json:"data"`
}

// applyLine updates st from an mpv property-change line and returns the event name.
func applyLine(line []byte, st *State) (string, error) {
	var m eventMsg
	if err := json.Unmarshal(line, &m); err != nil {
		return "", err
	}
	if m.Event == "property-change" {
		switch m.Name {
		case "time-pos":
			var f float64
			if json.Unmarshal(m.Data, &f) == nil {
				st.Position = int(f)
			}
		case "duration":
			var f float64
			if json.Unmarshal(m.Data, &f) == nil {
				st.Duration = int(f)
			}
		case "volume":
			var f float64
			if json.Unmarshal(m.Data, &f) == nil {
				st.Volume = int(f)
			}
		case "pause":
			var b bool
			if json.Unmarshal(m.Data, &b) == nil {
				st.Paused = b
			}
		case "media-title":
			var s string
			if json.Unmarshal(m.Data, &s) == nil {
				st.Title = s
			}
		}
	}
	return m.Event, nil
}

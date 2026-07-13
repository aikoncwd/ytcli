package tui

import (
	"github.com/AikonCWD/ytcli/internal/player"
	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/track"
)

// Ports keep the model testable with fakes and decouple it from concrete deps.
type playerPort interface {
	Load(url string) error
	TogglePause() error
	Seek(delta int) error
	SetVolume(v int) error
	State() player.State
	EndCh() <-chan struct{}
}

type searchPort interface {
	Search(query string, n int) ([]track.Track, error)
	Resolve(url string) ([]track.Track, error)
}

type storePort interface {
	AppendHistory(t track.Track) error
	ToggleFavorite(t track.Track) (bool, error)
	LoadHistory() ([]track.Track, error)
	LoadFavorites() ([]track.Track, error)
}

type mode int

const (
	modeCompact mode = iota
	modeExpanded
)

type tab int

const (
	tabQueue tab = iota
	tabSearch
	tabHistory
	tabFavorites
)

const (
	seekStep   = 10
	volumeStep = 5
	searchN    = 15
)

type Model struct {
	q      *queue.Queue
	player playerPort
	yt     searchPort
	store  storePort

	st      player.State
	vol     int
	prevVol int // for mute restore

	mode      mode
	tab       tab
	searching bool
	query     string
	cursor    int

	results   []track.Track
	history   []track.Track
	favorites []track.Track

	status string
	width  int
	height int
	quit   bool
}

func New(q *queue.Queue, p playerPort, yt searchPort, st storePort, vol int) Model {
	return Model{q: q, player: p, yt: yt, store: st, vol: vol, prevVol: vol, width: 46}
}

// View is a minimal placeholder so Model satisfies tea.Model (Init/Update/View).
// Full rendering (compact/expanded layouts using the Task 7 format helpers) is
// implemented in a later task; Task 8 only wires up state transitions.
func (m Model) View() string { return "" }

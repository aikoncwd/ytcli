// Package player runs an mpv process and drives it over mpv's JSON IPC on a
// Windows named pipe. Audio only; mpv resolves YouTube via yt-dlp internally.
package player

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	winio "github.com/Microsoft/go-winio"
)

const PipeName = `\\.\pipe\ytcli-mpv`

type State struct {
	Position int
	Duration int
	Volume   int
	Paused   bool
	Title    string
}

type Player struct {
	mpvPath  string
	pipeName string
	cmd      *exec.Cmd
	conn     net.Conn
	mu       sync.Mutex
	state    State
	endCh    chan struct{}
}

func New(mpvPath string) *Player {
	return &Player{mpvPath: mpvPath, pipeName: PipeName, endCh: make(chan struct{}, 1)}
}

func (p *Player) Start() error {
	p.cmd = exec.Command(p.mpvPath,
		"--no-video", "--idle=yes", "--no-terminal",
		"--input-ipc-server="+p.pipeName,
		"--volume=80",
	)
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("lanzando mpv: %w", err)
	}

	var conn net.Conn
	var err error
	for i := 0; i < 50; i++ { // mpv tarda un momento en crear el pipe
		timeout := 500 * time.Millisecond
		conn, err = winio.DialPipe(p.pipeName, &timeout)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("conectando al IPC de mpv: %w", err)
	}
	p.conn = conn

	for i, name := range []string{"time-pos", "duration", "volume", "pause", "media-title"} {
		if _, err := p.conn.Write(cmdObserve(i+1, name)); err != nil {
			return err
		}
	}
	go p.readLoop()
	return nil
}

func (p *Player) readLoop() {
	sc := bufio.NewScanner(p.conn)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		p.mu.Lock()
		ev, _ := applyLine(line, &p.state)
		p.mu.Unlock()
		if ev == "end-file" {
			select {
			case p.endCh <- struct{}{}:
			default:
			}
		}
	}
}

func (p *Player) send(b []byte) error {
	if p.conn == nil {
		return errors.New("player no iniciado")
	}
	_, err := p.conn.Write(b)
	return err
}

func (p *Player) Load(url string) error  { return p.send(cmdLoad(url)) }
func (p *Player) SetPaused(v bool) error { return p.send(cmdSetPause(v)) }
func (p *Player) Seek(d int) error       { return p.send(cmdSeek(d)) }
func (p *Player) SetVolume(v int) error  { return p.send(cmdSetVolume(v)) }

func (p *Player) TogglePause() error {
	p.mu.Lock()
	paused := p.state.Paused
	p.mu.Unlock()
	return p.SetPaused(!paused)
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) EndCh() <-chan struct{} { return p.endCh }

func (p *Player) Close() error {
	if p.conn != nil {
		p.conn.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

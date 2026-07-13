package deps

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveBinaryPreferPath(t *testing.T) {
	look := func(n string) (string, error) { return `C:\tools\` + n + ".exe", nil }
	exists := func(string) bool { return false }
	got, ok := resolveBinary("mpv", `C:\bin`, look, exists)
	if !ok || got != `C:\tools\mpv.exe` {
		t.Fatalf("resolve = %q,%v; want C:\\tools\\mpv.exe,true", got, ok)
	}
}

func TestResolveBinaryFallbackBinDir(t *testing.T) {
	look := func(string) (string, error) { return "", errors.New("not found") }
	exists := func(p string) bool { return strings.HasSuffix(p, `\bin\yt-dlp.exe`) }
	got, ok := resolveBinary("yt-dlp", `C:\bin`, look, exists)
	if !ok || !strings.HasSuffix(got, `\bin\yt-dlp.exe`) {
		t.Fatalf("resolve = %q,%v; want bin dir path,true", got, ok)
	}
}

func TestResolveBinaryMissing(t *testing.T) {
	look := func(string) (string, error) { return "", errors.New("no") }
	exists := func(string) bool { return false }
	if _, ok := resolveBinary("mpv", `C:\bin`, look, exists); ok {
		t.Fatal("missing binary should return false")
	}
}

func TestParseReleaseAssets(t *testing.T) {
	js := strings.NewReader(`{"assets":[
		{"name":"mpv-x86_64-20250101-git-abc.7z","browser_download_url":"https://x/a.7z"},
		{"name":"mpv-dev-x86_64-20250101.7z","browser_download_url":"https://x/dev.7z"}
	]}`)
	assets, err := parseReleaseAssets(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 || assets[0].Name != "mpv-x86_64-20250101-git-abc.7z" {
		t.Fatalf("assets = %+v", assets)
	}
}

func TestPickMpvAsset(t *testing.T) {
	assets := []asset{
		{Name: "mpv-dev-x86_64-20250101.7z", URL: "dev"},
		{Name: "mpv-x86_64-v3-20250101-git-abc.7z", URL: "v3"},
		{Name: "mpv-x86_64-20250101-git-abc.7z", URL: "good"},
	}
	got, err := pickMpvAsset(assets)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "good" {
		t.Fatalf("picked %q; want the plain x86_64 build", got.URL)
	}
}

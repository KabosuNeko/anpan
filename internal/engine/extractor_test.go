package engine

import (
	"regexp"
	"strings"
	"testing"
)

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int) *int           { return &v }

func TestExtractPortionsListsAllResolutions(t *testing.T) {
	meta := VideoMeta{
		Title:    "Test Video",
		Duration: floatPtr(120),
		Formats: []RawStream{
			{FormatID: "audio1", Acodec: "mp4a", ABR: floatPtr(128), Filesize: floatPtr(2_000_000)},
			{FormatID: "v4320", Vcodec: "av01", Height: intPtr(4320), FPS: floatPtr(60), TBR: floatPtr(25_000)},
			{FormatID: "v2160", Vcodec: "vp9", Height: intPtr(2160), FPS: floatPtr(60), DynamicRange: "HDR", TBR: floatPtr(15_000)},
			{FormatID: "v1440", Vcodec: "vp9", Height: intPtr(1440), FPS: floatPtr(60), TBR: floatPtr(8_000)},
			{FormatID: "v1080", Vcodec: "avc1", Height: intPtr(1080), FPS: floatPtr(60), TBR: floatPtr(4_000)},
			{FormatID: "v720", Vcodec: "avc1", Height: intPtr(720), FPS: floatPtr(30), TBR: floatPtr(2_000)},
			{FormatID: "v480", Vcodec: "avc1", Height: intPtr(480), FPS: floatPtr(30), TBR: floatPtr(1_000)},
			{FormatID: "v360", Vcodec: "avc1", Height: intPtr(360), FPS: floatPtr(30), TBR: floatPtr(600)},
			{FormatID: "v240", Vcodec: "avc1", Height: intPtr(240), FPS: floatPtr(30), TBR: floatPtr(350)},
			{FormatID: "v144", Vcodec: "avc1", Height: intPtr(144), FPS: floatPtr(30), TBR: floatPtr(150)},
		},
	}

	portions := ExtractPortions(meta, &ExtractPortionsOptions{
		VideoContainer: "mkv",
		AudioFormat:    "flac",
	})

	var videoPortions []Portion
	for _, p := range portions {
		if p.Kind == PortionKindVideo {
			videoPortions = append(videoPortions, p)
		}
	}

	if len(videoPortions) != 9 {
		t.Fatalf("expected 9 video portions, got %d", len(videoPortions))
	}

	expectedPrefixes := []string{
		"4320p60 · mkv",
		"2160p60 HDR · mkv",
		"1440p60 · mkv",
		"1080p60 · mkv",
		"720p · mkv",
		"480p · mkv",
		"360p · mkv",
		"240p · mkv",
		"144p · mkv",
	}

	for i, vp := range videoPortions {
		if !strings.HasPrefix(vp.Label, expectedPrefixes[i]) {
			t.Errorf("video portion %d: label %q does not have prefix %q", i, vp.Label, expectedPrefixes[i])
		}
		re := regexp.MustCompile(`~[\d.]+ (KB|MB|GB)`)
		if !re.MatchString(vp.Label) {
			t.Errorf("video portion %d: label %q missing estimated size tag", i, vp.Label)
		}
	}

	var audioPortions []Portion
	for _, p := range portions {
		if p.Kind == PortionKindAudio {
			audioPortions = append(audioPortions, p)
		}
	}
	if len(audioPortions) != 3 {
		t.Fatalf("expected 3 audio portions, got %d", len(audioPortions))
	}
	if !strings.HasPrefix(audioPortions[0].Label, "audio only · m4a (aac)") {
		t.Errorf("expected m4a (aac), got %s", audioPortions[0].Label)
	}
	if !strings.HasPrefix(audioPortions[1].Label, "audio only · flac") {
		t.Errorf("expected flac, got %s", audioPortions[1].Label)
	}
	if !strings.HasPrefix(audioPortions[2].Label, "audio only · mp3") {
		t.Errorf("expected mp3, got %s", audioPortions[2].Label)
	}
}

func TestCleanAlbumTitle(t *testing.T) {
	if got := CleanAlbumTitle("Album - Midnights"); got != "Midnights" {
		t.Errorf("CleanAlbumTitle('Album - Midnights') = %q", got)
	}
	if got := CleanAlbumTitle("album - Lover"); got != "Lover" {
		t.Errorf("CleanAlbumTitle('album - Lover') = %q", got)
	}
	if got := CleanAlbumTitle("Album: 1989"); got != "1989" {
		t.Errorf("CleanAlbumTitle('Album: 1989') = %q", got)
	}
	if got := CleanAlbumTitle("EP - The Secret"); got != "The Secret" {
		t.Errorf("CleanAlbumTitle('EP - The Secret') = %q", got)
	}
	if got := CleanAlbumTitle("Single - Anti-Hero"); got != "Anti-Hero" {
		t.Errorf("CleanAlbumTitle('Single - Anti-Hero') = %q", got)
	}
	if got := CleanAlbumTitle("Normal Playlist"); got != "Normal Playlist" {
		t.Errorf("CleanAlbumTitle('Normal Playlist') = %q", got)
	}
	if got := CleanAlbumTitle(""); got != "Playlist" {
		t.Errorf("CleanAlbumTitle('') = %q", got)
	}
}

func TestExtractPortionsVideoCodec(t *testing.T) {
	meta := VideoMeta{
		Title: "Multi Codec Video",
		Formats: []RawStream{
			{FormatID: "v1080_av1", Vcodec: "av01.0.08M.08", Height: intPtr(1080), TBR: floatPtr(2000)},
			{FormatID: "v1080_vp9", Vcodec: "vp09.00.41.08", Height: intPtr(1080), TBR: floatPtr(2000)},
			{FormatID: "v1080_avc", Vcodec: "avc1.640028", Height: intPtr(1080), TBR: floatPtr(2000)},
		},
	}

	// Test AV1 preference
	portionsAV1 := ExtractPortions(meta, &ExtractPortionsOptions{VideoCodec: "av1"})
	if len(portionsAV1) == 0 || !strings.Contains(portionsAV1[0].YtdlpArgs[1], "vcodec^=av01") {
		t.Errorf("expected av1 selector in args: %v", portionsAV1[0].YtdlpArgs)
	}

	// Test VP9 preference
	portionsVP9 := ExtractPortions(meta, &ExtractPortionsOptions{VideoCodec: "vp9"})
	if len(portionsVP9) == 0 || !strings.Contains(portionsVP9[0].YtdlpArgs[1], "vcodec^=vp09") {
		t.Errorf("expected vp9 selector in args: %v", portionsVP9[0].YtdlpArgs)
	}

	// Test AVC preference
	portionsAVC := ExtractPortions(meta, &ExtractPortionsOptions{VideoCodec: "avc"})
	if len(portionsAVC) == 0 || !strings.Contains(portionsAVC[0].YtdlpArgs[1], "vcodec^=avc1") {
		t.Errorf("expected avc selector in args: %v", portionsAVC[0].YtdlpArgs)
	}
}

func TestExtractPlaylistPortions(t *testing.T) {
	plPortions := ExtractPlaylistPortions(&ExtractPortionsOptions{
		VideoContainer: "webm",
		AudioFormat:    "m4a",
	})
	if len(plPortions) != 4 {
		t.Fatalf("expected 4 playlist portions, got %d", len(plPortions))
	}
	if !strings.Contains(plPortions[0].Label, "all tracks · m4a") {
		t.Errorf("expected m4a label, got %s", plPortions[0].Label)
	}
	if !strings.Contains(plPortions[1].Label, "all tracks · opus (original audio)") {
		t.Errorf("expected opus label, got %s", plPortions[1].Label)
	}
	if !strings.Contains(plPortions[2].Label, "all tracks · mp3") {
		t.Errorf("expected mp3 label, got %s", plPortions[2].Label)
	}
	if !strings.Contains(plPortions[3].Label, "all videos · webm") {
		t.Errorf("expected webm label, got %s", plPortions[3].Label)
	}
}

func TestExtractPortionsAudioOnly(t *testing.T) {
	meta := VideoMeta{
		Title:    "SoundCloud Track",
		Duration: floatPtr(180),
		Formats: []RawStream{
			{FormatID: "hls_mp3", Ext: "mp3", Acodec: "mp3", Vcodec: "none", ABR: floatPtr(128)},
			{FormatID: "hls_aac", Ext: "m4a", Acodec: "mp4a.40.2", Vcodec: "none", ABR: floatPtr(160)},
		},
	}

	portions := ExtractPortions(meta, nil)
	var videoPortions []Portion
	for _, p := range portions {
		if p.Kind == PortionKindVideo {
			videoPortions = append(videoPortions, p)
		}
	}
	if len(videoPortions) != 0 {
		t.Errorf("expected 0 video portions for soundcloud track, got %d", len(videoPortions))
	}
}

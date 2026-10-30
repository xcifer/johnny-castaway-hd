package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

func TestSoundID(t *testing.T) {
	want := map[int]int{
		1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 7: 7, 8: 8, 9: 9, 10: 10,
		12: 11, 14: 12, 15: 13, 16: 14, 17: 15, 18: 16, 19: 17, 20: 18,
		21: 19, 22: 20, 23: 21, 24: 22,
	}
	for n := -1; n <= 30; n++ {
		id, ok := SoundID(n)
		w, wok := want[n]
		if id != w || ok != wok {
			t.Errorf("SoundID(%d) = %d, %v; want %d, %v", n, id, ok, w, wok)
		}
	}
}

// wavFile builds a sound resource as the original stores them: 8-bit mono
// at 11025 Hz, padded to a multiple of 512 bytes.
func wavFile(samples []byte) []byte {
	var b []byte
	le32 := func(v int) { b = binary.LittleEndian.AppendUint32(b, uint32(v)) }
	le16 := func(v int) { b = binary.LittleEndian.AppendUint16(b, uint16(v)) }
	b = append(b, "RIFF"...)
	le32(36 + len(samples))
	b = append(b, "WAVEfmt "...)
	le32(16)
	le16(1)     // PCM
	le16(1)     // mono
	le32(11025) // rate
	le32(11025) // bytes per second
	le16(1)     // block align
	le16(8)     // bits
	b = append(b, "data"...)
	le32(len(samples))
	b = append(b, samples...)
	for len(b)%512 != 0 {
		b = append(b, 0x80^0x55) // padding must not be played
	}
	return b
}

func TestDecode(t *testing.T) {
	// A square wave: 100 samples at +64, 100 at -64, 8-bit unsigned.
	var in []byte
	for i := range 200 {
		in = append(in, byte(128+64-128*(i/100)))
	}
	pcm, err := Decode(wavFile(in))
	if err != nil {
		t.Fatal(err)
	}
	// 4 times the samples, each 2 channels of 2 bytes.
	if got, want := len(pcm), len(in)*4*4; got != want {
		t.Fatalf("%d bytes, want %d", got, want)
	}
	at := func(i int) (int16, int16) {
		return int16(binary.LittleEndian.Uint16(pcm[i*4:])), int16(binary.LittleEndian.Uint16(pcm[i*4+2:]))
	}
	// Away from the edges, the resampled wave keeps the original levels.
	for _, c := range []struct{ i, want int }{{100, 16384}, {200, 16384}, {300, 16384}, {500, -16384}, {600, -16384}, {700, -16384}} {
		l, r := at(c.i)
		if l != r {
			t.Errorf("sample %d: left %d, right %d", c.i, l, r)
		}
		if d := int(l) - c.want; d < -400 || d > 400 {
			t.Errorf("sample %d = %d, want about %d", c.i, l, c.want)
		}
	}
}

type fakeVoice struct {
	id  int
	log *[]string
}

func (v fakeVoice) Play()  { *v.log = append(*v.log, fmt.Sprintf("play %d", v.id)) }
func (v fakeVoice) Pause() { *v.log = append(*v.log, fmt.Sprintf("stop %d", v.id)) }
func (v fakeVoice) Rewind() error {
	*v.log = append(*v.log, fmt.Sprintf("rewind %d", v.id))
	return nil
}

func TestPlayer(t *testing.T) {
	var sounds []resource.Sound
	for id := 1; id <= 22; id++ {
		sounds = append(sounds, resource.Sound{ID: id, Data: wavFile([]byte{128, byte(id)})})
	}
	var calls, logs []string
	n := 0
	p, err := newPlayer(sounds, func([]byte) voice {
		n++
		return fakeVoice{n, &calls}
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Logf = func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }

	p.Play(3)
	p.Play(15) // sound 13 cuts sound 3 short
	p.Play(15) // and starts over
	p.Play(11) // no sound: sound 13 plays on
	p.Play(11)
	p.Play(13)
	p.Play(24)
	want := []string{
		"rewind 3", "play 3",
		"stop 3", "rewind 13", "play 13",
		"stop 13", "rewind 13", "play 13",
		"stop 13", "rewind 22", "play 22",
	}
	if !slices.Equal(calls, want) {
		t.Errorf("calls\n%q\nwant\n%q", calls, want)
	}
	wantLogs := []string{"PLAY_SAMPLE 11: no such sound", "PLAY_SAMPLE 13: no such sound"}
	if !slices.Equal(logs, wantLogs) {
		t.Errorf("logs %q, want %q", logs, wantLogs)
	}
}

func TestPlayerBadSound(t *testing.T) {
	_, err := newPlayer([]resource.Sound{{ID: 4, Data: []byte("RIFF....WAVE")}}, func([]byte) voice { return nil })
	if err == nil {
		t.Error("no error for a broken sound")
	}
}

// Every sound decodes, and every PLAY_SAMPLE in the scripts has a sound,
// except the one in GJCATCH2.TTM, for which no sound exists.
func TestGameData(t *testing.T) {
	const dir = "../../gamedata"
	exe, err := os.ReadFile(filepath.Join(dir, "SCRANTIC.SCR"))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("game data not present in gamedata/")
	}
	if err != nil {
		t.Fatal(err)
	}
	sounds, err := resource.Sounds(exe)
	if err != nil {
		t.Fatal(err)
	}
	have := map[int]bool{}
	for _, s := range sounds {
		pcm, err := Decode(s.Data)
		if err != nil {
			t.Errorf("sound %d: %v", s.ID, err)
			continue
		}
		// The data chunk's length, not the padded resource, sets the length.
		size := int(binary.LittleEndian.Uint32(s.Data[40:]))
		if string(s.Data[36:40]) != "data" || len(pcm) != size*16 {
			t.Errorf("sound %d: %d bytes of PCM for %d samples", s.ID, len(pcm), size)
		}
		have[s.ID] = true
	}

	a, err := resource.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	plays := 0
	for i := range a.Entries {
		e := &a.Entries[i]
		if e.Type() != "TTM" {
			continue
		}
		ttm, err := resource.DecodeTTM(e)
		if err != nil {
			t.Fatal(err)
		}
		for _, in := range ttm.Code {
			if in.Opcode != 0xC051 {
				continue
			}
			plays++
			if id, ok := SoundID(int(in.Args[0])); !ok || !have[id] {
				missing = append(missing, fmt.Sprintf("%s %d", e.Name, in.Args[0]))
			}
		}
	}
	if plays < 500 {
		t.Errorf("only %d PLAY_SAMPLE found", plays)
	}
	if want := []string{"GJCATCH2.TTM 11"}; !slices.Equal(missing, want) {
		t.Errorf("samples without a sound: %q, want %q", missing, want)
	}
}

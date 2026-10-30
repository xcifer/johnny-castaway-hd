// Package audio plays the original's sounds for the scripts' PLAY_SAMPLE.
//
// Like the original, it plays one sound at a time: a new sound stops the one
// playing. Recordings of the original show this wherever a loud sound is cut
// short by the next one.
package audio

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"

	"github.com/xcifer/johnny-castaway-hd/internal/resource"
)

// SampleRate is the output rate. The sounds are 11025 Hz, a quarter of it.
const SampleRate = 44100

// SoundID returns the resource ID in SCRANTIC.SCR of the sound that
// PLAY_SAMPLE n plays. Samples 11 and 13 have no sound.
//
// This is jc_reborn's numbering. Recordings of the original confirm it for
// samples 14, 15, 16, 22, 23 and 24, and agree with it for 19 and 21; see
// docs/formats.md.
func SoundID(sample int) (int, bool) {
	switch {
	case sample >= 1 && sample <= 10:
		return sample, true
	case sample == 12:
		return 11, true
	case sample >= 14 && sample <= 24:
		return sample - 2, true
	}
	return 0, false
}

// Decode converts a sound resource to 16-bit stereo PCM at SampleRate.
func Decode(data []byte) ([]byte, error) {
	s, err := wav.DecodeWithSampleRate(SampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(s)
}

// voice is one sound ready to play; *audio.Player is one.
type voice interface {
	Play()
	Pause()
	Rewind() error
}

// Player plays the sounds of PLAY_SAMPLE.
type Player struct {
	// Logf reports samples without a sound, each only once; nil uses the
	// log package.
	Logf func(format string, args ...any)

	voices  map[int]voice // by resource ID
	playing voice
	warned  map[int]bool
}

// NewPlayer prepares the sounds for playing. It starts the audio output, so
// call it only once.
func NewPlayer(sounds []resource.Sound) (*Player, error) {
	ctx := audio.NewContext(SampleRate)
	return newPlayer(sounds, func(pcm []byte) voice { return ctx.NewPlayerFromBytes(pcm) })
}

func newPlayer(sounds []resource.Sound, newVoice func(pcm []byte) voice) (*Player, error) {
	p := &Player{voices: map[int]voice{}, warned: map[int]bool{}}
	for _, s := range sounds {
		pcm, err := Decode(s.Data)
		if err != nil {
			return nil, fmt.Errorf("sound %d: %w", s.ID, err)
		}
		p.voices[s.ID] = newVoice(pcm)
	}
	return p, nil
}

// Play stops the sound playing, if any, and starts the one of PLAY_SAMPLE n
// from its beginning.
func (p *Player) Play(sample int) {
	id, ok := SoundID(sample)
	v := p.voices[id]
	if !ok || v == nil {
		if !p.warned[sample] {
			p.warned[sample] = true
			p.logf("PLAY_SAMPLE %d: no such sound", sample)
		}
		return
	}
	if p.playing != nil {
		p.playing.Pause()
	}
	if err := v.Rewind(); err != nil {
		p.logf("sound %d: %v", id, err)
		return
	}
	v.Play()
	p.playing = v
}

func (p *Player) logf(format string, args ...any) {
	if p.Logf != nil {
		p.Logf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

// Package speech defines speech-to-text and text-to-speech abstractions.
// The ElevenLabs implementations live in internal/speech/elevenlabs (M5).
package speech

import (
	"context"
	"io"
	"iter"
	"time"
)

type Word struct {
	Text       string
	Start, End time.Duration
	Confidence float64
}

type Transcript struct {
	Text     string
	Language string // BCP-47 primary subtag as detected: "it", "en", "ru"
	Words    []Word
	Duration time.Duration
}

type STTOptions struct {
	MIMEType     string // e.g. "audio/webm", "audio/wav"
	LanguageHint string // empty = auto-detect
	// Keyterms is the domain vocabulary (config/keyterms.txt), loaded and
	// validated by LoadKeyterms. nil sends none (eval baseline). Keyterms are
	// billed extra by ElevenLabs.
	Keyterms []string
}

type STT interface {
	Transcribe(ctx context.Context, audio io.Reader, opts STTOptions) (*Transcript, error)
}

type TTSOptions struct {
	VoiceID  string
	Language string // answer language; multilingual model is told explicitly
	Format   string // e.g. "mp3_44100_128", "pcm_16000"
	// PreviousText/NextText let providers keep prosody continuous when text is
	// synthesised sentence by sentence.
	PreviousText string
	NextText     string
}

type TTS interface {
	// SynthesizeStream yields audio chunks as they are produced.
	// Breaking out of the loop cancels the upstream request.
	SynthesizeStream(ctx context.Context, text string, opts TTSOptions) iter.Seq2[[]byte, error]
}

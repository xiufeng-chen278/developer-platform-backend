package audio

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/dh1tw/gosamplerate"
)

const (
	InSampleRate  = 24000
	OutSampleRate = 48000
	Channels      = 1
	Ratio         = float64(OutSampleRate) / float64(InSampleRate)
)

type Processor struct {
	src   *gosamplerate.Src
	srcMu sync.Mutex
}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) InitSRC() error {
	p.srcMu.Lock()
	defer p.srcMu.Unlock()
	if p.src != nil {
		return nil
	}
	s, err := gosamplerate.New(gosamplerate.SRC_SINC_MEDIUM_QUALITY, Channels, 2048)
	if err != nil {
		return fmt.Errorf("init SRC: %w", err)
	}
	p.src = &s
	return nil
}

func (p *Processor) Resample(pcm24k []byte, final bool) ([]byte, error) {
	if len(pcm24k) == 0 && !final {
		return nil, nil
	}
	if err := p.InitSRC(); err != nil {
		return nil, err
	}
	if len(pcm24k)%2 != 0 {
		pcm24k = pcm24k[:len(pcm24k)-1]
	}
	n := len(pcm24k) / 2
	in := make([]float32, n)
	for i := 0; i < n; i++ {
		s := int16(binary.LittleEndian.Uint16(pcm24k[2*i:]))
		in[i] = float32(s) / 32768.0
	}
	p.srcMu.Lock()
	outFloats, err := p.src.Process(in, Ratio, final)
	p.srcMu.Unlock()
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(outFloats)*2)
	for i, f := range outFloats {
		if f > 1 {
			f = 1
		} else if f < -1 {
			f = -1
		}
		s := int16(f * 32767.0)
		binary.LittleEndian.PutUint16(out[2*i:], uint16(s))
	}
	return out, nil
}

package streamer

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/ebml-go/webm"
	"github.com/gopxl/beep"
	"github.com/kkdai/youtube/v2"
	"layeh.com/gopus"
)

func NewYTWebMOpusStreamer(video *youtube.Video) (beep.StreamSeekCloser, beep.Format, error) {
	if video == nil {
		return nil, beep.Format{}, fmt.Errorf("video es nil, no se pudo obtener información de YouTube")
	}

	formats := video.Formats.Type(`audio/webm; codecs="opus"`)
	if len(formats) == 0 {
		return nil, beep.Format{}, fmt.Errorf("no se encontró formato opus/webm para este video")
	}

	format := formats[0]
	stream, _, err := yt.GetStream(video, &format)
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("error al obtener stream: %w", err)
	}

	data := make([]byte, format.ContentLength)
	if _, err := stream.Read(data); err != nil && err != io.EOF {
		return nil, beep.Format{}, fmt.Errorf("error al leer stream: %w", err)
	}

	var w webm.WebM
	reader, err := webm.Parse(bytes.NewReader(data), &w)
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("error al parsear webm: %w", err)
	}

	audioTrack := w.FindFirstAudioTrack()
	form := beep.Format{
		SampleRate:  beep.SampleRate(audioTrack.SamplingFrequency),
		NumChannels: int(audioTrack.Channels),
		Precision:   2,
	}

	decoder, err := gopus.NewDecoder(int(audioTrack.SamplingFrequency), int(audioTrack.Channels))
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("error creando decoder: %w", err)
	}

	frameSize := float64(audioTrack.Channels) * 60.0 * audioTrack.SamplingFrequency / 1000

	return &pcmStreamer{
		frameSize: int(frameSize),
		decoder:   decoder,
		packets:   reader.Chan,
		reader:    reader,
		duration:  video.Duration,
	}, form, nil
}

type pcmStreamer struct {
	pcm       []int16
	pcmIdx    int
	frameSize int

	decoder *gopus.Decoder
	packets <-chan webm.Packet
	reader  *webm.Reader

	duration time.Duration

	pos int

	err error

	closed bool
}

var _ beep.StreamSeekCloser = (*pcmStreamer)(nil)

func (s *pcmStreamer) Err() error {
	err := s.err
	s.err = nil
	return err
}

func (s *pcmStreamer) Seek(n int) error {
	s.reader.Seek(SampleRate.D(n))
	s.pos = n
	s.pcm = nil
	s.pcmIdx = 0
	return s.err
}

func (s *pcmStreamer) Position() int {
	return s.pos
}

func (s *pcmStreamer) Len() int {
	return SampleRate.N(s.duration)
}

func (s *pcmStreamer) Close() error {
	if s.closed {
		return ErrAlreadyClosed
	}
	s.closed = true
	return nil
}

func (s *pcmStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if s.closed {
		s.err = ErrClosed
		return
	}
	if s.pos >= SampleRate.N(s.duration) {
		s.err = io.EOF
		return 0, false
	}
	for n < len(samples) {
		if s.pcmIdx >= len(s.pcm) {
			select {
			case packet := <-s.packets:
				s.pcm, s.err = s.decoder.Decode(packet.Data, int(s.frameSize), false)
				s.pcmIdx = 0
			default:
			}
		}
		if len(s.pcm) <= s.pcmIdx || s.err != nil {
			//audio done
			fmt.Println("audio done")
			break
		}
		for ; n < len(samples); n++ {
			samples[n][0] = float64(s.pcm[s.pcmIdx]) / 32767
			samples[n][1] = float64(s.pcm[s.pcmIdx+1]) / 32767
			s.pcmIdx += 2
			if s.pcmIdx >= len(s.pcm) {
				break
			}
		}
	}

	s.pos += len(samples)

	return n, true
}

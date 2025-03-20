package domain

import (
	"io"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

func StreamMP3FromReader(reader io.Reader) {
	// Декодируем MP3 из переданного потока
	decodedMp3, err := mp3.NewDecoder(reader)
	if err != nil {
		panic("mp3.NewDecoder failed: " + err.Error())
	}

	// Настройка oto контекста
	op := &oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	}
	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		panic("oto.NewContext failed: " + err.Error())
	}
	<-readyChan

	player := otoCtx.NewPlayer(decodedMp3)
	player.Play()

	// Ждём окончания воспроизведения
	for player.IsPlaying() {
		time.Sleep(time.Millisecond * 100)
	}

	if err = player.Close(); err != nil {
		panic("player.Close failed: " + err.Error())
	}
}

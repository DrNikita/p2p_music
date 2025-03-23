package domain

import (
	"context"
	"io"
	"log/slog"
	"p2p-music/domain/playlist"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

const (
	songStreamingProtocol = "/song/stream/1.0.0"
)

type DomainManager struct {
	h      host.Host
	gp     *playlist.GlobalPlaylist
	dht    *dht.IpfsDHT
	logger *slog.Logger
}

func NewDomainManager() *DomainManager {
	return nil
}

func (dm *DomainManager) PromoteSong(ctx context.Context, filePath string, song playlist.Song) error {
	songCID, err := generateCIDFromFile(filePath)
	if err != nil {
		return err
	}

	if err := dm.dht.Provide(ctx, songCID, false); err != nil {
		return err
	}

	if err := dm.gp.AdvertiseSong(song); err != nil {
		return err
	}

	return nil
}

func (dm *DomainManager) Register(ctx context.Context) {
	dm.h.SetStreamHandler(songStreamingProtocol, dm.streamSong)
}

func (dm *DomainManager) streamSong(s network.Stream) {}

func (dm *DomainManager) receiveSongStream(ctx context.Context, songName string) {
	// find song CID
	// find song provider
	// receive stream
}

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

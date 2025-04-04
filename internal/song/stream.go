package song

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"p2p-music/config"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	songStreamingProtocol = "/song/stream/1.1.0"
)

type Store interface {
	SaveFilePath(context.Context, cid.Cid, string) error

	FindFilePath(context.Context, cid.Cid) (string, error)
}

type SongTableManager interface {
	AdvertiseSong(Song) error

	Search(string) (Song, error)

	SearchWithParams(Song) (Song, error)

	RegisterSongTableHandlers(context.Context, host.Host)
}

type SongManager struct {
	h         host.Host
	songTable SongTableManager
	dht       *dht.IpfsDHT
	store     Store
	config    *config.Config
	logger    *slog.Logger
}

func NewSongManager(h host.Host, songTable SongTableManager, dht *dht.IpfsDHT, store Store, config *config.Config, logger *slog.Logger) *SongManager {
	return &SongManager{
		h:         h,
		songTable: songTable,
		dht:       dht,
		store:     store,
		config:    config,
		logger:    logger,
	}
}

func (dm *SongManager) PromoteSong(ctx context.Context, song Song, songFilePath string) error {
	if err := dm.store.SaveFilePath(ctx, song.CID, songFilePath); err != nil {
		dm.logger.Error("Failed to save song file path", "err", err)
		return PromoteSongError{
			errMsg: err.Error(),
		}
	}
	dm.logger.Info("Successfully saved song file path", "path", songFilePath)

	if err := dm.dht.Provide(ctx, song.CID, true); err != nil {
		dm.logger.Error("Failed to provide song", "err", err)
		return PromoteSongError{
			errMsg: err.Error(),
		}
	}

	if err := dm.songTable.AdvertiseSong(song); err != nil {
		dm.logger.Error("Failed to advertise song", "err", err)
		return PromoteSongError{
			errMsg: err.Error(),
		}
	}

	return nil
}

func (dm *SongManager) FindSongProviders(ctx context.Context, song Song) ([]peer.AddrInfo, error) {
	nonSelfProviders := make([]peer.AddrInfo, 0)

	providers, err := dm.dht.FindProviders(ctx, song.CID)
	if err != nil {
		return nil, err
	}

	for _, provider := range providers {
		if provider.ID != dm.h.ID() {
			nonSelfProviders = append(nonSelfProviders, provider)
		}
	}

	return nonSelfProviders, nil
}

// TODO: promote song after receving
func (dm *SongManager) ReceiveSongStream(ctx context.Context, song Song, targetPeerID peer.ID) (string, error) {
	stream, err := dm.h.NewStream(context.Background(), targetPeerID, songStreamingProtocol)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	_, err = stream.Write([]byte(song.Title + "\n")) // Wrire separator
	if err != nil {
		return "", err
	}

	songNewFilePath := fmt.Sprintf("%s/%s.%s", dm.config.MusicPath, song.SongNameWithoutFormat(), song.SongFormat())
	outFile, err := os.Create(songNewFilePath)
	if err != nil {
		dm.logger.Error("Failed to create song file", "err", err)
		return "", err
	}
	defer outFile.Close()

	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			_, writeErr := outFile.Write(buf[:n])
			if writeErr != nil {
				return "", writeErr
			}
		}
		if err == io.EOF {
			dm.logger.Info("Audio stream ended")
			break
		} else if err != nil {
			return "", err
		}
	}

	return songNewFilePath, nil
}

func (dm *SongManager) RegisterSongStreamingProtocols(ctx context.Context) {
	dm.h.SetStreamHandler(songStreamingProtocol, func(s network.Stream) {
		defer s.Close()
		err := dm.streamSong(ctx, s)
		if err != nil {
			dm.logger.Error("Failed to stream song", "err", err)
			// s.Reset()
			return
		}
		dm.logger.Info("Song streaming success")
	})
}

func (dm *SongManager) streamSong(ctx context.Context, s network.Stream) error {
	// Читаем имя запрашиваемого файла
	reader := bufio.NewReader(s)
	songTitle, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Ошибка чтения запроса:", err)
		return err
	}
	songTitle = strings.TrimSpace(songTitle)

	song, err := dm.songTable.Search(songTitle)
	if err != nil {
		dm.logger.Error("Failed to find file", "song title", songTitle, "err", err)
		return err
	}

	songPath, err := dm.store.FindFilePath(ctx, song.CID)
	if err != nil {
		return err
	}

	file, err := os.Open(songPath)
	if err != nil {
		dm.logger.Error("Failed to open file", "err", err)
		return err
	}
	defer file.Close()

	// Стримим аудиофайл чанками
	buf := make([]byte, 4096)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, writeErr := s.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}

// TODO: unused
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

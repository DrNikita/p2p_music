package domain

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"p2p-music/config"
	"p2p-music/internal/model"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	songStreamingProtocol = "/song/stream/1.1.0"
)

type SongTableManager interface {
	AdvertiseSong(song model.Song) error
	Search(song string) (model.Song, error)
	RegisterSongTableHandlers(ctx context.Context, h host.Host)
}

type DomainManager struct {
	h         host.Host
	songTable SongTableManager
	dht       *dht.IpfsDHT
	config    *config.Config
	logger    *slog.Logger
}

func NewDomainManager(h host.Host, songTable SongTableManager, dht *dht.IpfsDHT, config *config.Config, logger *slog.Logger) *DomainManager {
	return &DomainManager{
		h:         h,
		songTable: songTable,
		dht:       dht,
		config:    config,
		logger:    logger,
	}
}

func (dm *DomainManager) PromoteSong(ctx context.Context, song model.Song) error {
	if err := dm.dht.Provide(ctx, song.CID, true); err != nil {
		return err
	}

	if err := dm.songTable.AdvertiseSong(song); err != nil {
		return err
	}

	return nil
}

func (dm *DomainManager) RegisterProtocols(ctx context.Context) {
	dm.h.SetStreamHandler(songStreamingProtocol, dm.streamSong)
}

func (dm *DomainManager) streamSong(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)

	// Читаем имя запрашиваемого файла
	filename, err := reader.ReadString('\n')
	if err != nil {
		s.Reset()
		fmt.Println("Ошибка чтения запроса:", err)
		return
	}
	filename = strings.TrimSpace(filename)

	fmt.Println("Передаю файл:", filename)

	// Открываем файл для чтения
	file, err := os.Open(filename)
	if err != nil {
		s.Reset()
		fmt.Println("Файл не найден:", filename)
		return
	}
	defer file.Close()

	// Стримим аудиофайл чанками
	buf := make([]byte, 4096)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, writeErr := s.Write(buf[:n])
			if writeErr != nil {
				return
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return
		}
	}
}

func (dm *DomainManager) ReceiveSongStream(ctx context.Context, song model.Song, targetPeerID peer.ID) error {
	stream, err := dm.h.NewStream(context.Background(), targetPeerID, songStreamingProtocol)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Отправляем имя файла
	_, err = stream.Write([]byte(song.Title + "\n")) // Разделитель для чтения
	if err != nil {
		return err
	}

	songName := songTitleParser(song.Title)
	if songName == "" {

	}

	// Создаем файл для записи
	outFile, err := os.Create(fmt.Sprintf("%s/%s", dm.config.MusicPath, song.Title))
	if err != nil {
		return err
	}
	defer outFile.Close()

	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			_, writeErr := outFile.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
		}
		if err == io.EOF {
			dm.logger.Info("Audio stream ended")
			break
		} else if err != nil {
			return err
		}
	}

	return nil
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

func (dm *DomainManager) FindSongProviders(ctx context.Context, song model.Song) ([]peer.AddrInfo, error) {
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

func songTitleParser(title string) string {
	titleParts := strings.Split(title, ".")
	if len(titleParts) == 0 {
		return ""
	}
	return titleParts[len(titleParts)-1]
}

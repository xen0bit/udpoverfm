package main

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"fmt"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/sigurn/crc8"
)

const (
	screenWidth  = 640
	screenHeight = 480

	sampleRate = 32000
)

var (
	playerBarColor     = color.RGBA{0x80, 0x80, 0x80, 0xff}
	playerCurrentColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

func init() {
}

type musicType int

const (
	typeOgg musicType = iota
)

var (
	//go:embed dtmf/0.ogg
	dtmf0 []byte
	//go:embed dtmf/1.ogg
	dtmf1 []byte
	//go:embed dtmf/2.ogg
	dtmf2 []byte
	//go:embed dtmf/3.ogg
	dtmf3 []byte
	//go:embed dtmf/4.ogg
	dtmf4 []byte
	//go:embed dtmf/5.ogg
	dtmf5 []byte
	//go:embed dtmf/6.ogg
	dtmf6 []byte
	//go:embed dtmf/7.ogg
	dtmf7 []byte
	//go:embed dtmf/8.ogg
	dtmf8 []byte
	//go:embed dtmf/9.ogg
	dtmf9 []byte
	//go:embed dtmf/a.ogg
	dtmfA []byte
	//go:embed dtmf/b.ogg
	dtmfB []byte
	//go:embed dtmf/c.ogg
	dtmfC []byte
	//go:embed dtmf/d.ogg
	dtmfD []byte
	//go:embed dtmf/star.ogg
	dtmfStar []byte
	//go:embed dtmf/pound.ogg
	dtmfPound []byte

	keyToTone map[string][]byte

	inputString = "hi im remy"
	backupQueue []string
	dtmfQueue   []string
)

// Player represents the current audio state.
type Player struct {
	game         *Game
	audioContext *audio.Context
	audioPlayer  *audio.Player
	current      time.Duration
	total        time.Duration
	seBytes      []byte
	seCh         chan []byte
	volume128    int
	musicType    musicType
}

func NewPlayer(game *Game, audioContext *audio.Context, dtmfCode string) (*Player, error) {
	type audioStream interface {
		io.ReadSeeker
		Length() int64
	}

	const bytesPerSample = 4 // TODO: This should be defined in audio package

	var s audioStream

	var err error
	s, err = vorbis.Decode(audioContext, bytes.NewReader(keyToTone[dtmfCode]))
	if err != nil {
		return nil, err
	}

	p, err := audioContext.NewPlayer(s)
	if err != nil {
		return nil, err
	}
	player := &Player{
		game:         game,
		audioContext: audioContext,
		audioPlayer:  p,
		total:        time.Second * time.Duration(s.Length()) / bytesPerSample / sampleRate,
		volume128:    128,
		seCh:         make(chan []byte),
	}
	if player.total == 0 {
		player.total = 1
	}

	player.audioPlayer.Play()
	return player, nil
}

func (p *Player) Close() error {
	return p.audioPlayer.Close()
}

func (p *Player) update() error {
	return nil
}

func (p *Player) draw(screen *ebiten.Image) {
	// Draw the debug message.
	msg := dtmfQueue
	ebitenutil.DebugPrint(screen, strings.Join(msg, " "))
}

type Game struct {
	musicPlayer   *Player
	musicPlayerCh chan *Player
	errCh         chan error
}

func NewGame() (*Game, error) {
	audioContext := audio.NewContext(sampleRate)

	g := &Game{
		musicPlayerCh: make(chan *Player),
		errCh:         make(chan error),
	}

	var key string
	key, dtmfQueue = dtmfQueue[0], dtmfQueue[1:]
	m, err := NewPlayer(g, audioContext, key)
	if err != nil {
		return nil, err
	}

	g.musicPlayer = m
	return g, nil
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		dtmfQueue = backupQueue
	}
	fmt.Println(dtmfQueue)

	select {
	case p := <-g.musicPlayerCh:
		g.musicPlayer = p
	case err := <-g.errCh:
		return err
	default:
	}

	if g.musicPlayer != nil {
		if !g.musicPlayer.audioPlayer.IsPlaying() {
			if len(dtmfQueue) > 0 {
				var key string
				key, dtmfQueue = dtmfQueue[0], dtmfQueue[1:]
				m, err := NewPlayer(g, g.musicPlayer.audioContext, key)
				if err != nil {
					panic(err)
				}

				g.musicPlayer = m
			}
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.musicPlayer != nil {
		g.musicPlayer.draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("UDP over FM Radio (DTMF)")

	//Set up Map
	keyToTone = make(map[string][]byte)
	keyToTone["0"] = dtmf0
	keyToTone["1"] = dtmf1
	keyToTone["2"] = dtmf2
	keyToTone["3"] = dtmf3
	keyToTone["4"] = dtmf4
	keyToTone["5"] = dtmf5
	keyToTone["6"] = dtmf6
	keyToTone["7"] = dtmf7
	keyToTone["8"] = dtmf8
	keyToTone["9"] = dtmf9
	keyToTone["a"] = dtmfA
	keyToTone["b"] = dtmfB
	keyToTone["c"] = dtmfC
	keyToTone["d"] = dtmfD
	keyToTone["*"] = dtmfStar
	keyToTone["#"] = dtmfPound

	hexInput := hex.EncodeToString([]byte(inputString))
	for i := 0; i < len(hexInput); i++ {
		key := string(hexInput[i])
		if key == "e" {
			dtmfQueue = append(dtmfQueue, "*")
		} else if key == "f" {
			dtmfQueue = append(dtmfQueue, "#")
		} else {
			dtmfQueue = append(dtmfQueue, string(hexInput[i]))
		}
	}

	table := crc8.MakeTable(crc8.CRC8_MAXIM)
	crc := crc8.Checksum([]byte(inputString), table)
	hexCrc := strconv.FormatInt(int64(crc), 16)
	dtmfQueue = append(dtmfQueue, string(hexCrc[0]))
	dtmfQueue = append(dtmfQueue, string(hexCrc[1]))
	backupQueue = dtmfQueue

	g, err := NewGame()
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

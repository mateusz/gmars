package main

import (
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
	"os"

	"github.com/bobertlo/gmars/pkg/mars"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

const (
	tileSize         = 6
	defaultSpeedStep = 6
)

type Game struct {
	sim       mars.ReportingSimulator
	rec       mars.StateRecorder
	running   bool
	speedStep int
	counter   int
}

var (
	mplusFaceSource *text.GoTextFaceSource

	//go:embed assets/tiles_6.png
	tiles_png []byte

	tilesImage *ebiten.Image

	speeds = []int{-64, -32, -16, -8, -4, -2, 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384}
)

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s

	img, _, err := image.Decode(bytes.NewReader(tiles_png))
	if err != nil {
		log.Fatal(err)
	}
	tilesImage = ebiten.NewImageFromImage(img)

}

func (g *Game) slowDown() {
	g.speedStep--
	if g.speedStep < 0 {
		g.speedStep = 0
	}
}

func (g *Game) speedUp() {
	g.speedStep++
	if g.speedStep >= len(speeds) {
		g.speedStep = len(speeds) - 1
	}
}

func (g *Game) handleInput() {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.running = !g.running
	} else if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.sim.Reset()
		g.sim.SpawnWarrior(0, 0)
		g.sim.SpawnWarrior(1, mars.Address(rand.Intn(7000)+200))
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.slowDown()
	} else if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.speedUp()
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.running = false
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if g.running {
			g.running = false
		} else {
			for i := 0; i < speeds[g.speedStep]; i++ {
				g.runCycle()
			}
		}
	}
}

func (g *Game) runCycle() {
	if g.sim.WarriorLivingCount() > 1 {
		g.sim.RunCycle()
	}
}

func (g *Game) Update() error {
	speed := speeds[g.speedStep]

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return errors.New("game ended by player")
	}

	if g.running {
		if speed < 0 {
			if g.counter%speed == 0 {
				g.runCycle()
			}
		} else {
			for i := 0; i < speeds[g.speedStep]; i++ {
				g.runCycle()
			}
		}
	}

	g.handleInput()

	g.counter++

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	scales := make([]ebiten.ColorScale, 3)
	scales[0].Scale(1, 1, 1, 1)
	scales[1].Scale(1, 1, 0, 1)
	scales[2].Scale(0, 1, 1, 1)

	w := tilesImage.Bounds().Dx()
	tileXCount := w / tileSize

	const xCount = screenWidth / tileSize

	for i := 0; i < int(g.sim.CoreSize()); i++ {
		state, color := g.rec.GetMemState(mars.Address(i))

		if state == mars.CoreEmpty {
			continue
		}
		t := int(state)

		op := &ebiten.DrawImageOptions{ColorScale: scales[color+1]}
		op.GeoM.Translate(float64((i%xCount)*tileSize), float64((i/xCount)*tileSize))

		sx := (t % tileXCount) * tileSize
		sy := (t / tileXCount) * tileSize
		screen.DrawImage(tilesImage.SubImage(image.Rect(sx, sy, sx+tileSize, sy+tileSize)).(*ebiten.Image), op)
	}

	// Draw info
	msg := fmt.Sprintf("FPS: %0.2f", ebiten.ActualTPS())
	op := &text.DrawOptions{}
	op.GeoM.Translate(560, 460)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   10,
	}, op)

	msg = fmt.Sprintf("Cycle: %05d (%dx)", g.sim.CycleCount(), speeds[g.speedStep])
	op = &text.DrawOptions{}
	op.GeoM.Translate(30, 460)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   10,
	}, op)

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	use88Flag := flag.Bool("8", false, "Enforce ICWS'88 rules")
	sizeFlag := flag.Int("s", 8000, "Size of core")
	procFlag := flag.Int("p", 8000, "Max. Processes")
	cycleFlag := flag.Int("c", 80000, "Cycles until tie")
	lenFlag := flag.Int("l", 100, "Max. warrior length")
	fixedFlag := flag.Int("F", 0, "fixed position of warrior #2")
	// roundFlag := flag.Int("r", 1, "Rounds to play")
	debugFlag := flag.Bool("debug", false, "Dump verbose reporting of simulator state")
	flag.Parse()

	coresize := mars.Address(*sizeFlag)
	processes := mars.Address(*procFlag)
	cycles := mars.Address(*cycleFlag)
	length := mars.Address(*lenFlag)

	var mode mars.SimulatorMode
	if *use88Flag {
		mode = mars.ICWS88
	} else {
		mode = mars.ICWS94
	}
	config := mars.NewQuickConfig(mode, coresize, processes, cycles, length)

	args := flag.Args()

	if len(args) < 2 || len(args) > 2 {
		fmt.Println("only 2 warrior battles supported")
		os.Exit(1)
	}

	w1file, err := os.Open(args[0])
	if err != nil {
		fmt.Printf("error opening warrior file '%s': %s\n", args[0], err)
		os.Exit(1)
	}
	defer w1file.Close()
	w1data, err := mars.ParseLoadFile(w1file, config)
	if err != nil {
		fmt.Printf("error parsing warrior file '%s': %s\n", args[0], err)
		os.Exit(1)
	}
	w1file.Close()

	w2file, err := os.Open(args[1])
	if err != nil {
		fmt.Printf("error opening warrior file '%s': %s\n", args[1], err)
		os.Exit(1)
	}
	defer w1file.Close()
	w2data, err := mars.ParseLoadFile(w2file, config)
	if err != nil {
		fmt.Printf("error parsing warrior file '%s': %s\n", args[1], err)
	}
	w1file.Close()

	sim, err := mars.NewReportingSimulator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating sim: %s", err)
	}
	if *debugFlag {
		sim.AddReporter(mars.NewDebugReporter(sim))
	}
	rec := mars.NewStateRecorder(sim)
	sim.AddReporter(rec)

	w2start := *fixedFlag
	if w2start == 0 {
		minStart := 2 * config.Length
		maxStart := config.CoreSize - config.Length - 1
		startRange := maxStart - minStart
		w2start = rand.Intn(int(startRange)+1) + int(minStart)
	}

	sim.AddWarrior(&w1data)
	sim.AddWarrior(&w2data)

	sim.SpawnWarrior(0, 0)
	sim.SpawnWarrior(1, mars.Address(w2start))

	game := &Game{
		sim:       sim,
		rec:       *rec,
		speedStep: defaultSpeedStep,
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(fmt.Sprintf("gMARS - '%s' vs '%s'", w1data.Name, w2data.Name))
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

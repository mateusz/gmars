package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"log"
	"os"

	"github.com/bobertlo/gmars"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 2 * 640
	screenHeight = 2 * 480
)

const (
	tileSize         = 15
	defaultSpeedStep = 6
)

var (
	mplusFaceSource *text.GoTextFaceSource

	//go:embed assets/tiles_15.png
	tiles_png []byte

	tilesImage *ebiten.Image

	speeds = []int{-64, -32, -16, -8, -4, -2, 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192}
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

func main() {
	use88Flag := flag.Bool("8", false, "Enforce ICWS'88 rules")
	sizeFlag := flag.Int("s", 8000, "Size of core")
	procFlag := flag.Int("p", 8000, "Max. Processes")
	cycleFlag := flag.Int("c", 80000, "Cycles until tie")
	lenFlag := flag.Int("l", 100, "Max. warrior length")
	fixedFlag := flag.Int("F", 0, "fixed position of warrior #2")
	// roundFlag := flag.Int("r", 1, "Rounds to play")
	showReadFlag := flag.Bool("showread", false, "display reads in the visualizer")
	debugFlag := flag.Bool("debug", false, "Dump verbose reporting of simulator state")
	flag.Parse()

	coresize := gmars.Address(*sizeFlag)
	processes := gmars.Address(*procFlag)
	cycles := gmars.Address(*cycleFlag)
	length := gmars.Address(*lenFlag)
	fixed := gmars.Address(*fixedFlag)

	var mode gmars.SimulatorMode
	if *use88Flag {
		mode = gmars.ICWS88
	} else {
		mode = gmars.ICWS94
	}
	config := gmars.NewQuickConfig(mode, coresize, processes, cycles, length, fixed)

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
	w1data, err := gmars.ParseLoadFile(w1file, config)
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
	w2data, err := gmars.ParseLoadFile(w2file, config)
	if err != nil {
		fmt.Printf("error parsing warrior file '%s': %s\n", args[1], err)
	}
	w1file.Close()

	sim, err := gmars.NewReportingSimulator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating sim: %s", err)
	}
	if *debugFlag {
		sim.AddReporter(gmars.NewDebugReporter(sim))
	}
	rec := gmars.NewStateRecorder(sim, 10)
	rec.SetRecordRead(*showReadFlag)
	sim.AddReporter(rec)

	sim.AddWarrior(&w1data)
	sim.AddWarrior(&w2data)

	sim.SpawnWarrior(0, 0)
	sim.SpawnWarrior(1, config.GetW2Start())

	game := NewGame(config, sim, rec, defaultSpeedStep)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(fmt.Sprintf("gMARS - '%s' vs '%s'", w1data.Name, w2data.Name))
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

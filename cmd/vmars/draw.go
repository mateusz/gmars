package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/bobertlo/gmars"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	SpriteBackground = iota
	SpriteShelfEmpty
	SpriteShelfFull
	SpriteTrap
	SpriteTowerRWall
	SpriteTower
	SpriteTowerLWall
	SpriteTowerAlone
	SpriteHead
	SpriteHeadActive
	SpriteLast
)
const xCount = screenWidth / tileSize

func (g *Game) precomputeSprites() {
	g.spriteCache = make([]*ebiten.Image, SpriteLast)
	sprnum := SpriteBackground

	tileXCount := tilesImage.Bounds().Dx() / tileSize
	for y := 0; y < tileXCount; y++ {
		for x := 0; x < tileXCount; x++ {
			sx := x * tileSize
			sy := y * tileSize
			rect := image.Rect(sx, sy, sx+tileSize, sy+tileSize)

			subImage := tilesImage.SubImage(rect).(*ebiten.Image)
			g.spriteCache[sprnum] = subImage

			sprnum++
			if sprnum >= SpriteLast {
				goto stop
			}
		}
	}
stop:
}

func (g *Game) BlitSpriteWithHue(screen *ebiten.Image, sprnum int, a gmars.Address, hue, sat, val float64) {
	var c colorm.ColorM
	c.ChangeHSV(hue, sat, val)
	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(float64((a%xCount)*tileSize), float64((a/xCount)*tileSize))
	colorm.DrawImage(screen, g.spriteCache[sprnum], c, op)
}

func (g *Game) BlitSprite(screen *ebiten.Image, sprnum int, a gmars.Address) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((a%xCount)*tileSize), float64((a/xCount)*tileSize))
	screen.DrawImage(g.spriteCache[sprnum], op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	for i := 0; i < int(g.sim.CoreSize()); i++ {
		a := gmars.Address(i)
		state, _ := g.rec.GetMemState(gmars.Address(i))
		if state == gmars.CoreRead {
			g.BlitSprite(screen, SpriteShelfFull, a)
		} else if state == gmars.CoreWritten {
			g.BlitSprite(screen, SpriteShelfEmpty, a)
		} else if state == gmars.CoreExecuted {
			g.BlitSprite(screen, SpriteTowerAlone, a)
		} else if state == gmars.CoreDecremented {
			g.BlitSprite(screen, SpriteTrap, a)
		} else if state == gmars.CoreIncremented {
			g.BlitSprite(screen, SpriteTrap, a)
		}
	}

	wc := g.sim.WarriorCount()
	hueIncr := (2 * math.Pi) / float64(wc)
	hue := 2 * math.Pi / 4.0
	for i := 0; i < int(wc); i++ {
		w := g.sim.GetWarrior(i)
		if w == nil || !w.Alive() {
			continue
		}

		valDecr := 1.0 / float64(len(w.Queue()))
		val := 1.0
		for _, pc := range w.Queue()[1:] {
			g.BlitSpriteWithHue(screen, SpriteHead, pc, hue, 1.0, val)
			val -= valDecr
		}

		pc, _ := w.NextPC()
		g.BlitSpriteWithHue(screen, SpriteHeadActive, pc, hue, 1.0, 1.0)

		hue += hueIncr
	}

	// Draw info
	msg := fmt.Sprintf("FPS: %0.2f", ebiten.ActualTPS())
	op := &text.DrawOptions{}
	op.GeoM.Translate(585+640, 465+480)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   10,
	}, op)

	msg = fmt.Sprintf("Cycle: %05d (%dx)", g.sim.CycleCount(), speeds[g.speedStep])
	op = &text.DrawOptions{}
	op.GeoM.Translate(5, 465+480)
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, msg, &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   10,
	}, op)

	// draw results if finished
	if g.finished {
		w1a := g.sim.GetWarrior(0).Alive()
		w2a := g.sim.GetWarrior(1).Alive()

		if w1a || w2a {

			var msg string
			op = &text.DrawOptions{}
			op.GeoM.Translate(115, 480+465)
			if w1a && w2a {
				msg = "tie"
			} else if w1a {
				msg = fmt.Sprintf("%s wins", g.sim.GetWarrior(0).Name())
			} else if w2a {
				msg = fmt.Sprintf("%s wins", g.sim.GetWarrior(1).Name())
			}
			text.Draw(screen, msg, &text.GoTextFace{
				Source: mplusFaceSource,
				Size:   10,
			}, op)
		}
	} else if !g.running {
		op = &text.DrawOptions{}
		op.GeoM.Translate(115, 480+465)
		text.Draw(screen, "PAUSED", &text.GoTextFace{
			Source: mplusFaceSource,
			Size:   10,
		}, op)
	}

}

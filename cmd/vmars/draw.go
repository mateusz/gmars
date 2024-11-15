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
	SpriteEye
	SpriteLast
)

const xCount = screenWidth / tileSize

func (g *Game) precomputeSprites() {
	g.spriteCache = make([][]*ebiten.Image, SpriteLast)
	sprnum := SpriteBackground

	tileXCount := tilesImage.Bounds().Dx() / tileSize
	for y := 0; y < tileXCount; y++ {
		for x := 0; x < tileXCount; x++ {
			sx := x * tileSize
			sy := y * tileSize
			rect := image.Rect(sx, sy, sx+tileSize, sy+tileSize)

			subImage := tilesImage.SubImage(rect).(*ebiten.Image)

			g.spriteCache[sprnum] = make([]*ebiten.Image, len(g.hues))
			for i, hue := range g.hues {
				var c colorm.ColorM
				c.ChangeHSV(hue/180.0*math.Pi, 1.0, 1.0)
				g.spriteCache[sprnum][i] = ebiten.NewImage(tileSize, tileSize)
				colorm.DrawImage(g.spriteCache[sprnum][i], subImage, c, nil)
			}

			sprnum++
			if sprnum >= SpriteLast {
				goto stop
			}
		}
	}
stop:
}

func (g *Game) BlitSpriteSV(screen *ebiten.Image, sprnum int, a gmars.Address, hue int, sat, val float64) {
	var offsetY float64
	if sprnum != SpriteHead && sprnum != SpriteHeadActive {
		offsetY = tileSize / 3.0
	}
	var c colorm.ColorM
	c.ChangeHSV(0.0, sat, val)
	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(float64((a%xCount)*tileSize), float64((a/xCount)*tileSize)+offsetY)
	colorm.DrawImage(screen, g.spriteCache[sprnum][hue], c, op)
}

func (g *Game) BlitSpriteAlpha(screen *ebiten.Image, sprnum int, a gmars.Address, hue int, alpha float32) {
	var offsetY float64
	if sprnum != SpriteHead && sprnum != SpriteHeadActive {
		offsetY = tileSize / 3.0
	}
	var scale ebiten.ColorScale
	scale.ScaleAlpha(alpha)
	op := &ebiten.DrawImageOptions{ColorScale: scale}
	op.GeoM.Translate(float64((a%xCount)*tileSize), float64((a/xCount)*tileSize)+offsetY)
	screen.DrawImage(g.spriteCache[sprnum][hue], op)
}

func (g *Game) BlitSprite(screen *ebiten.Image, sprnum int, a gmars.Address, hue int) {
	var offsetY float64
	if sprnum != SpriteHead && sprnum != SpriteHeadActive {
		offsetY = tileSize / 3.0
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((a%xCount)*tileSize), float64((a/xCount)*tileSize)+offsetY)
	screen.DrawImage(g.spriteCache[sprnum][hue], op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	for i := 0; i < int(g.sim.CoreSize()); i++ {
		a := gmars.Address(i)
		mem := g.sim.GetMem(a)
		curState, curWar := g.rec.GetMemState(gmars.Address(i))
		prevState, prevWar := g.rec.GetMemStateN(gmars.Address(i), 1)

		if curState == gmars.CoreWritten && mem.Op == gmars.DAT {
			g.BlitSprite(screen, SpriteShelfEmpty, a, curWar)
		} else if curState == gmars.CoreWritten && mem.Op != gmars.DAT {
			g.BlitSprite(screen, SpriteTowerAlone, a, curWar)
		} else if curState == gmars.CoreRead && prevState == gmars.CoreWritten && prevWar == curWar {
			g.BlitSprite(screen, SpriteShelfFull, a, curWar)
		} else if curState == gmars.CoreRead && prevState == gmars.CoreRead && prevWar == curWar {
			g.BlitSprite(screen, SpriteShelfFull, a, curWar)
		} else if curState == gmars.CoreRead && prevState == gmars.CoreExecuted && prevWar == curWar {
			g.BlitSprite(screen, SpriteShelfFull, a, curWar)
		} else if curState == gmars.CoreRead && prevState == gmars.CoreIncremented && prevWar == curWar {
			g.BlitSprite(screen, SpriteShelfFull, a, curWar)
		} else if curState == gmars.CoreRead && prevState == gmars.CoreDecremented && prevWar == curWar {
			g.BlitSprite(screen, SpriteShelfFull, a, curWar)
		} else if curState == gmars.CoreExecuted && mem.Op != gmars.DAT {
			g.BlitSprite(screen, SpriteTowerAlone, a, curWar)
		} else if curState == gmars.CoreDecremented {
			g.BlitSprite(screen, SpriteTrap, a, curWar)
		} else if curState == gmars.CoreIncremented {
			g.BlitSprite(screen, SpriteTrap, a, curWar)
		} else if curState == gmars.CoreRead {
			g.BlitSprite(screen, SpriteEye, a, curWar)
		}
	}

	wc := g.sim.WarriorCount()
	for i := 0; i < int(wc); i++ {
		w := g.sim.GetWarrior(i)
		if w == nil || !w.Alive() {
			continue
		}

		valDecr := 0.9 / float64(len(w.Queue()))
		val := 1.0
		for _, pc := range w.Queue()[1:] {
			g.BlitSpriteAlpha(screen, SpriteHead, pc, i, float32(val*val))
			val -= valDecr
		}

		pc, _ := w.NextPC()
		g.BlitSprite(screen, SpriteHeadActive, pc, i)
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

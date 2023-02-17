package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/exp/slices"
)

const (
	paddingX     = 22
	paddingY     = 5
	spaceX       = 4
	spaceY       = 8
	scale        = 4
	shooterSpeed = 1
	bulletSpeed  = 2.5
	bulletOffset = 7

	characterCols   = 11
	characterRows   = 10
	characterWidth  = 12
	characterHeight = 8

	pixelsWidth  = paddingX*2 + characterCols*(12+spaceX) - spaceX
	pixelsHeight = paddingY*2 + characterRows*(9+spaceY)
)

const (
	octopus    = "octopus"
	octopusAlt = "octopus_alt"
	crab       = "crab"
	crabAlt    = "crab_alt"
	squid      = "squid"
	squidAlt   = "squid_alt"
	shooter    = "shooter"
)

var characters = map[string][]string{
	octopus: {
		"    ####    ",
		" ########## ",
		"############",
		"###  ##  ###",
		"############",
		"  ###  ###  ",
		" ##  ##  ## ",
		"  ##    ##  ",
	},
	octopusAlt: {
		"    ####    ",
		" ########## ",
		"############",
		"###  ##  ###",
		"############",
		"   ##  ##   ",
		"  ## ## ##  ",
		"##        ##",
	},
	crab: {
		"  #     #   ",
		"   #   #    ",
		"  #######   ",
		" ## ### ##  ",
		"########### ",
		"########### ",
		"# #     # # ",
		"   ## ##    ",
	},
	crabAlt: {
		"  #     #   ",
		"#  #   #  # ",
		"# ####### # ",
		"### ### ### ",
		"########### ",
		" #########  ",
		"  #     #   ",
		" #       #  ",
	},
	squid: {
		"     ##     ",
		"    ####    ",
		"   ######   ",
		"  ## ## ##  ",
		"  ########  ",
		"   # ## #   ",
		"  #      #  ",
		"   #    #   ",
	},
	squidAlt: {
		"     ##     ",
		"    ####    ",
		"   ######   ",
		"  ## ## ##  ",
		"  ########  ",
		"    #  #    ",
		"   # ## #   ",
		"  # #  # #  ",
	},
	shooter: {
		"       #       ",
		"      ###      ",
		"      ###      ",
		" ############# ",
		"###############",
		"###############",
		"###############",
		"###############",
	},
}

type Character struct {
	Row       int
	Col       int
	X         float64
	Y         float64
	character string
}

func (c Character) Inside(x, y float64) bool {
	cx, cy := c.X, c.Y
	cw, ch := float64(characterWidth), float64(characterHeight)

	if x < cx || x > cx+cw {
		return false
	}
	if y < cy || y > cy+ch {
		return false
	}
	return true
}

type Point struct {
	X float64
	Y float64
}

type Bullet struct {
	X    float64
	Y    float64
	Velo float64
}

type Invaders struct {
	step         int
	keys         []ebiten.Key
	pressed      map[ebiten.Key]bool
	bullets      []Point
	enemies      []Character
	enemyVelo    float64
	enemyBullets []Bullet
	shooter      Character
	shooterDead  bool
}

func (inv *Invaders) Init() {
	inv.step = 0
	inv.shooterDead = false

	// listen for keys
	inv.keys = []ebiten.Key{
		ebiten.KeyLeft,
		ebiten.KeyRight,
		ebiten.KeySpace,
	}

	inv.pressed = map[ebiten.Key]bool{}
	for _, k := range inv.keys {
		inv.pressed[k] = false
	}

	// create shooter
	_, py := enemyPosition(0, characterRows)
	inv.shooter = Character{
		X:         pixelsWidth/2 - characterWidth/2,
		Y:         py,
		character: shooter,
	}
	inv.bullets = nil

	// create enemies
	inv.enemies = nil
	for y, character := range []string{squid, crab, crab, octopus, octopus} {
		for x := 0; x < characterCols; x++ {
			px, py := enemyPosition(x, y)
			inv.enemies = append(inv.enemies, Character{
				Row:       y,
				Col:       x,
				X:         px,
				Y:         py,
				character: character,
			})
		}
	}
	inv.enemyBullets = nil
	inv.enemyVelo = 2
}

func (inv *Invaders) Update() error {
	if inv.shooterDead {
		inv.Init()
		return nil
	}

	inv.step++
	inv.updateKeys()
	inv.updateShooter()
	inv.updateBullets()
	inv.updateEnemies()
	inv.updateCollisions()
	inv.updateEnemyBullets()
	return nil
}

func (inv *Invaders) Draw(screen *ebiten.Image) {
	for _, enemy := range inv.enemies {
		drawCharacter(screen, enemy.X, enemy.Y, enemy.character, color.White)
	}
	for _, b := range inv.bullets {
		drawBullet(screen, float64(int(b.X))+bulletOffset, float64(int(b.Y/8)*8))
	}
	for _, b := range inv.enemyBullets {
		drawEnemyBullet(screen, float64(int(b.X)), float64(int(b.Y/2)*2))
	}
	if !inv.shooterDead {
		drawCharacter(screen, float64(int(inv.shooter.X)), inv.shooter.Y, inv.shooter.character, color.RGBA{52, 255, 0, 255})
	}
}

func (inv *Invaders) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return pixelsWidth * scale, pixelsHeight * scale
}

func (inv *Invaders) updateKeys() {
	for _, k := range inv.keys {
		if inpututil.IsKeyJustPressed(k) {
			inv.pressed[k] = true
		}
		if inpututil.IsKeyJustReleased(k) {
			inv.pressed[k] = false
		}
	}
}

func (inv *Invaders) updateShooter() {
	if inv.pressed[ebiten.KeyLeft] {
		inv.shooter.X -= shooterSpeed
	}
	if inv.pressed[ebiten.KeyRight] {
		inv.shooter.X += shooterSpeed
	}
	inv.shooter.X = bounds(inv.shooter.X, 0, pixelsWidth-characterWidth)
}

func (inv *Invaders) updateBullets() {
	if inv.pressed[ebiten.KeySpace] && len(inv.bullets) == 0 {
		inv.bullets = append(inv.bullets, Point{X: inv.shooter.X, Y: inv.shooter.Y})
	}
	var newBullets []Point
	for _, b := range inv.bullets {
		if b.Y < 0 {
			continue
		}
		newBullets = append(newBullets, Point{b.X, b.Y - bulletSpeed})
	}
	inv.bullets = newBullets
}

func (inv *Invaders) updateEnemies() {
	if inv.step%30 != 0 {
		return
	}

	var max float64
	var min float64 = math.MaxFloat64

	for _, enemy := range inv.enemies {
		max = math.Max(enemy.X+12, max)
		min = math.Min(enemy.X, min)
	}

	var stepY bool
	if min-paddingX == -16 {
		inv.enemyVelo = 2
		stepY = true
	}
	if pixelsWidth-paddingX-max == -16 {
		inv.enemyVelo = -2
		stepY = true
	}

	var newEnemies []Character
	for _, enemy := range inv.enemies {
		switch enemy.character {
		case octopus:
			enemy.character = octopusAlt
		case octopusAlt:
			enemy.character = octopus
		case crab:
			enemy.character = crabAlt
		case crabAlt:
			enemy.character = crab
		case squid:
			enemy.character = squidAlt
		case squidAlt:
			enemy.character = squid
		}

		enemy.X += inv.enemyVelo
		if stepY {
			enemy.Y += 8
		}
		newEnemies = append(newEnemies, enemy)
	}

	inv.enemies = newEnemies
}

func (inv *Invaders) updateCollisions() {
	hit := func(enemy Character) (int, bool) {
		for bi, bullet := range inv.bullets {
			if enemy.Inside(bullet.X+bulletOffset, bullet.Y) {
				return bi, true
			}
		}
		return 0, false
	}

	var newEnemies []Character
	for _, enemy := range inv.enemies {
		if bullet, ok := hit(enemy); ok {
			inv.bullets = slices.Delete(inv.bullets, bullet, bullet+1)
			continue
		}
		newEnemies = append(newEnemies, enemy)
	}

	inv.enemies = newEnemies
}

func (inv *Invaders) updateEnemyBullets() {
	mod := len(inv.enemies)*2 + 20
	if mod == 0 {
		mod = 1
	}
	if inv.step%mod == 0 {
		enemies := bottomEnemies(inv.enemies)
		if len(enemies) > 0 {
			enemy := enemies[rand.Intn(len(enemies))]
			inv.enemyBullets = append(inv.enemyBullets, Bullet{X: enemy.X + 5, Y: enemy.Y + 16, Velo: rand.Float64() + 1})
		}
	}

	var newBullets []Bullet
	for _, bullet := range inv.enemyBullets {
		bullet.Y += bullet.Velo
		if bullet.Y > pixelsHeight {
			continue
		}
		newBullets = append(newBullets, bullet)
	}
	inv.enemyBullets = newBullets

	for _, bullet := range inv.enemyBullets {
		if inv.shooter.Inside(bullet.X, bullet.Y) {
			inv.shooterDead = true
		}
	}
}

func bottomEnemies(enemies []Character) []Character {
	transposed := make([][]int, characterCols)
	for i := 0; i < characterCols; i++ {
		transposed[i] = make([]int, 5)
		for j := 0; j < 5; j++ {
			transposed[i][j] = -1
		}
	}
	for i, enemy := range enemies {
		transposed[enemy.Col][enemy.Row] = i
	}

	var bottom []Character
	for _, row := range transposed {
		b := -1
		for _, col := range row {
			if col != -1 {
				b = col
			}
		}
		if b == -1 {
			continue
		}
		bottom = append(bottom, enemies[b])
	}
	return bottom
}

func drawCharacter(screen *ebiten.Image, x, y float64, character string, color color.Color) {
	for cy, row := range characters[character] {
		for cx, col := range row {
			if col == '#' {
				ebitenutil.DrawRect(screen, x*scale+float64(cx)*scale, y*scale+float64(cy)*scale, scale, scale, color)
			}
		}
	}
}

func drawBullet(screen *ebiten.Image, x, y float64) {
	ebitenutil.DrawRect(screen, x*scale, y*scale, scale, scale*6, color.RGBA{255, 0, 0, 255})
}

func drawEnemyBullet(screen *ebiten.Image, x, y float64) {
	ebitenutil.DrawRect(screen, x*scale, y*scale, scale, scale*6, color.White)
}

func enemyPosition(x, y int) (float64, float64) {
	return paddingX + float64(x)*(12+spaceX), paddingY + float64(y)*(8+spaceY)
}

func bounds(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func main() {
	ebiten.SetWindowSize(pixelsWidth*scale, pixelsHeight*scale)
	ebiten.SetWindowTitle("Hello, World!")

	inv := &Invaders{}
	inv.Init()
	if err := ebiten.RunGame(inv); err != nil {
		log.Fatal(err)
	}
}

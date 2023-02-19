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
	paddingX = 22
	paddingY = 5
	spaceX   = 4
	spaceY   = 8
	scale    = 4

	shooterSpeed         = 1
	bulletSpeed          = 4
	bulletOffset         = 7
	enemyBulletSpeed     = 1.33
	enemyBulletSpeedFast = 1.66

	characterCols   = 11
	characterRows   = 10
	characterWidth  = 12
	characterHeight = 8
	shooterWidth    = 15

	pixelsWidth  = paddingX*2 + characterCols*(12+spaceX) - spaceX
	pixelsHeight = paddingY*2 + characterRows*(9+spaceY)
)

const (
	octopus     = "octopus"
	crab        = "crab"
	squid       = "squid"
	shooter     = "shooter"
	explosion   = "explosion"
	bulletZap   = "bullet_zap"
	bulletCross = "bullet_cross"
)

var characters = map[string][][]string{
	octopus: {
		{
			"    ####    ",
			" ########## ",
			"############",
			"###  ##  ###",
			"############",
			"  ###  ###  ",
			" ##  ##  ## ",
			"  ##    ##  ",
		},
		{
			"    ####    ",
			" ########## ",
			"############",
			"###  ##  ###",
			"############",
			"   ##  ##   ",
			"  ## ## ##  ",
			"##        ##",
		},
	},
	crab: {
		{
			"  #     #   ",
			"   #   #    ",
			"  #######   ",
			" ## ### ##  ",
			"########### ",
			"########### ",
			"# #     # # ",
			"   ## ##    ",
		},
		{
			"  #     #   ",
			"#  #   #  # ",
			"# ####### # ",
			"### ### ### ",
			"########### ",
			" #########  ",
			"  #     #   ",
			" #       #  ",
		},
	},
	squid: {
		{
			"     ##     ",
			"    ####    ",
			"   ######   ",
			"  ## ## ##  ",
			"  ########  ",
			"   # ## #   ",
			"  #      #  ",
			"   #    #   ",
		},
		{
			"     ##     ",
			"    ####    ",
			"   ######   ",
			"  ## ## ##  ",
			"  ########  ",
			"    #  #    ",
			"   # ## #   ",
			"  # #  # #  ",
		},
	},
	shooter: {
		{
			"       #       ",
			"      ###      ",
			"      ###      ",
			" ############# ",
			"###############",
			"###############",
			"###############",
			"###############",
			"               ",
			"               ",
			"               ",
			"               ",
			"               ",
			"               ",
			"               ",
		},
	},
	explosion: {
		{
			"     #      ",
			" #   #  #   ",
			"  #     #  #",
			"   #   #  # ",
			"##          ",
			"          ##",
			" #  #   #   ",
			"#  #     #  ",
			"   #  #   # ",
			"      #     ",
		},
	},
	bulletZap: {
		{
			" # ",
			"  #",
			" # ",
			"#  ",
			" # ",
		},
		{
			" # ",
			"#  ",
			" # ",
			"  #",
			" # ",
		},
	},
	bulletCross: {
		{
			" # ",
			"###",
			" # ",
			" #",
			" # ",
		},
	},
}

type Character struct {
	Row       int
	Col       int
	X         float64
	Y         float64
	character string
	dying     int
}

func (c Character) Character(step int) []string {
	if c.dying != 0 {
		return characters[explosion][0]
	}

	cc := characters[c.character]
	return cc[step/30%len(cc)]
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
	X         float64
	Y         float64
	Velo      float64
	character string
}

func (b Bullet) Character(step int) []string {
	cc := characters[b.character]
	return cc[step/20%len(cc)]
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
	restartWait  int
}

func (inv *Invaders) Init() {
	inv.step = 0
	inv.shooterDead = false
	inv.restartWait = 0

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
		inv.restartWait++
		if inv.restartWait == 120 {
			inv.Init()
		}
		return nil
	}

	inv.step++
	inv.updateKeys()
	inv.updateShooter()
	inv.launchBullet()
	inv.moveBullets()
	inv.moveEnemies()
	inv.updateCollisions()
	inv.updateEnemyBullets()
	return nil
}

func (inv *Invaders) Draw(screen *ebiten.Image) {
	for _, enemy := range inv.enemies {
		drawCharacter(screen, floor(enemy.X), floor(enemy.Y), enemy.Character(inv.step), color.White)
	}
	for _, b := range inv.bullets {
		drawBullet(screen, floor(b.X)+bulletOffset, floor(b.Y/4)*4)
	}
	for _, b := range inv.enemyBullets {
		drawCharacter(screen, floor(b.X), floor(b.Y/4)*4, b.Character(inv.step), color.White)
	}

	drawCharacter(screen, floor(inv.shooter.X), floor(inv.shooter.Y), inv.shooter.Character(inv.step), color.RGBA{52, 255, 0, 255})

	if inv.shooterDead {
		ebitenutil.DrawRect(screen, 0, 0, scale, pixelsHeight*scale, color.RGBA{255, 0, 0, 255})
		ebitenutil.DrawRect(screen, 0, 0, pixelsWidth*scale, scale, color.RGBA{255, 0, 0, 255})
		ebitenutil.DrawRect(screen, pixelsWidth*scale-scale, 0, scale, pixelsHeight*scale, color.RGBA{255, 0, 0, 255})
		ebitenutil.DrawRect(screen, 0, 0, pixelsWidth*scale, scale, color.RGBA{255, 0, 0, 255})
		ebitenutil.DrawRect(screen, 0, pixelsHeight*scale-scale, pixelsWidth*scale, scale, color.RGBA{255, 0, 0, 255})
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
	inv.shooter.X = bounds(inv.shooter.X, 0, pixelsWidth-shooterWidth)
}

func (inv *Invaders) moveBullets() {
	var new []Point
	for _, b := range inv.bullets {
		if b.Y < 0 {
			continue
		}
		new = append(new, Point{b.X, b.Y - bulletSpeed})
	}
	inv.bullets = new
}

func (inv *Invaders) launchBullet() {
	if inv.pressed[ebiten.KeySpace] && len(inv.bullets) == 0 {
		inv.bullets = append(inv.bullets, Point{X: inv.shooter.X, Y: inv.shooter.Y})
	}
}

func (inv *Invaders) moveEnemies() {
	if inv.step%30 != 0 {
		return
	}

	var (
		max = -math.MaxFloat64
		min = math.MaxFloat64
	)
	for _, enemy := range inv.enemies {
		max = math.Max(enemy.X+characterWidth, max)
		min = math.Min(enemy.X, min)
	}

	var (
		mostLeft    = min-paddingX == -16
		mostRight   = pixelsWidth-paddingX-max == -16
		moveNextRow = mostLeft || mostRight
	)

	if mostLeft {
		if inv.enemyVelo == 0 {
			inv.enemyVelo = 2
		} else {
			inv.enemyVelo = 0
		}
	}

	if mostRight {
		if inv.enemyVelo == 0 {
			inv.enemyVelo = -2
		} else {
			inv.enemyVelo = 0
		}
	}

	for i := range inv.enemies {
		inv.enemies[i].X += inv.enemyVelo
		if moveNextRow && inv.enemyVelo == 0 {
			inv.enemies[i].Y += 8
		}
	}
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
		if enemy.dying == 2 {
			continue
		} else if enemy.dying > 0 && inv.step%30 == 0 {
			enemy.dying++
		}
		if bullet, ok := hit(enemy); ok {
			inv.bullets = slices.Delete(inv.bullets, bullet, bullet+1)
			enemy.dying = 1
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

			speed := float64(enemyBulletSpeed)
			if len(enemies) <= 8 {
				speed = float64(enemyBulletSpeedFast)
			}

			inv.enemyBullets = append(inv.enemyBullets, Bullet{
				X:         enemy.X + 4,
				Y:         enemy.Y + 16,
				Velo:      speed,
				character: []string{bulletZap, bulletCross}[rand.Intn(2)],
			})
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

func drawCharacter(screen *ebiten.Image, x, y float64, character []string, color color.Color) {
	for cy, row := range character {
		for cx, col := range row {
			if col == '#' {
				ebitenutil.DrawRect(screen, x*scale+float64(cx)*scale, y*scale+float64(cy)*scale, scale, scale, color)
			}
		}
	}
}

func drawBullet(screen *ebiten.Image, x, y float64) {
	ebitenutil.DrawRect(screen, x*scale, y*scale, scale, scale*4, color.White)
}

func drawEnemyBullet(screen *ebiten.Image, x, y float64) {
	ebitenutil.DrawRect(screen, x*scale, y*scale, scale, scale, color.White)
	ebitenutil.DrawRect(screen, (x+1)*scale, (y+1)*scale, scale, scale, color.White)
	ebitenutil.DrawRect(screen, x*scale, (y+2)*scale, scale, scale, color.White)
	ebitenutil.DrawRect(screen, (x-1)*scale, (y+3)*scale, scale, scale, color.White)
	ebitenutil.DrawRect(screen, x*scale, (y+4)*scale, scale, scale, color.White)
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

func floor(f float64) float64 {
	return float64(int(f))
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

package main

import (
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/nsf/termbox-go"
)

type direction int8

type gameState int8

const (
	up    = direction(-1)
	down  = direction(1)
	right = direction(2)
	left  = direction(-2)

	playing  = gameState(1)
	gameOver = gameState(2)
	exit     = gameState(3)

	wallColor = 7
)

type node struct {
	x int
	y int
	d direction
}

type snake struct {
	nodes     []*node
	d         direction
	potential int
}

func (s *snake) head() *node {
	return s.nodes[len(s.nodes)-1]
}

func (s *snake) tail() []*node {
	return s.nodes[0 : len(s.nodes)-1]
}

func (s *snake) move() {
	head := s.head()
	head.d = s.d

	var newHead *node
	if s.potential > 0 {
		s.potential--
		newHead = &node{}
		s.nodes = append(s.nodes, newHead)
	} else {
		newHead = s.nodes[0]
		s.nodes = append(s.nodes[1:], newHead)
	}

	newHead.d = s.d
	newHead.x = head.x
	newHead.y = head.y
	switch newHead.d {
	case up:
		newHead.y--
	case down:
		newHead.y++
	case left:
		newHead.x--
	case right:
		newHead.x++
	}
}

func (s *snake) redirect(d direction) {
	if (s.head().d + d) != 0 {
		s.d = d
	}
}

type food struct {
	x int
	y int
}

type game struct {
	snake  *snake
	ticker *time.Ticker
	events chan termbox.Event
	state  gameState
	food   *food
	score  int
	width  int
	height int
}

func (g *game) tick() {
	if g.state == playing {
		g.snake.move()
		g.consolidate()
	}
}

func (g *game) draw() {
	termbox.Clear(termbox.ColorDefault, termbox.Attribute(1))

	// draw snake
	for _, n := range g.snake.nodes {
		termbox.SetCell(n.x*2, n.y, ' ', termbox.ColorDefault, termbox.Attribute(2))
		termbox.SetCell(n.x*2+1, n.y, ' ', termbox.ColorDefault, termbox.Attribute(2))
	}

	// draw food
	termbox.SetCell(g.food.x*2, g.food.y, ' ', termbox.ColorDefault, termbox.Attribute(4))
	termbox.SetCell(g.food.x*2+1, g.food.y, ' ', termbox.ColorDefault, termbox.Attribute(4))

	// draw borders
	sx, sy := g.width, g.height
	for x := 0; x < sx*2; x++ {
		termbox.SetCell(x, 0, ' ', termbox.ColorDefault, termbox.Attribute(wallColor))
		termbox.SetCell(x, sy-1, ' ', termbox.ColorDefault, termbox.Attribute(wallColor))
	}
	for y := 0; y < sy; y++ {
		termbox.SetCell(0, y, ' ', termbox.ColorDefault, termbox.Attribute(wallColor))
		termbox.SetCell(1, y, ' ', termbox.ColorDefault, termbox.Attribute(wallColor))
		termbox.SetCell(sx*2-1, y, ' ', termbox.ColorDefault, termbox.Attribute(wallColor))
		termbox.SetCell(sx*2, y, ' ', termbox.ColorDefault, termbox.Attribute(wallColor))
	}

	// draw score
	puts(" Score: "+strconv.Itoa(g.score)+" ", 3, 0)

	termbox.Flush()
}

func (g *game) consolidate() {
	sx, sy := g.width, g.height
	head := g.snake.head()
	for _, n := range g.snake.tail() {
		if (n.x == head.x) && (n.y == head.y) {
			g.state = gameOver
		}
	}

	if (head.x == 0) || (head.x == sx-1) || (head.y == 0) || (head.y == sy-1) {
		g.state = gameOver
	}

	if (head.x == g.food.x) && (head.y == g.food.y) {
		g.snake.potential = 10
		g.score++
		x, y := freeSpot(g.width, g.height, g.snake.nodes)
		g.food.x = x
		g.food.y = y
	}
}

func (g *game) loop() {
	for {

		switch g.state {

		case playing:
			select {
			case <-g.ticker.C:
				g.tick()
				g.draw()
			case ch := <-g.events:
				switch ch.Key {
				case termbox.KeyCtrlC:
					g.state = exit
				case termbox.KeyArrowUp:
					g.snake.redirect(up)
				case termbox.KeyArrowDown:
					g.snake.redirect(down)
				case termbox.KeyArrowRight:
					g.snake.redirect(right)
				case termbox.KeyArrowLeft:
					g.snake.redirect(left)
				}
			default:
			}

		case gameOver:
			cputs("Game Over! Press space to start again or ESC to exit.")
			select {
			case ch := <-g.events:
				switch ch.Key {
				case termbox.KeyEsc:
					fallthrough
				case termbox.KeyCtrlC:
					return
				case termbox.KeySpace:
					g.snake = newSnake(1, right)
					g.score = 0
					g.state = playing
				}
			}
		case exit:
			return
		}
	}
}

func newSnake(size, d direction) *snake {
	const (
		initX         = 5
		initY         = 5
		initPotential = 5
	)
	var nodes = make([]*node, size)
	for i := range nodes {
		nodes[i] = &node{initX + i, initY, right}
	}
	return &snake{nodes: nodes, potential: initPotential, d: d}
}

func newGame(events chan termbox.Event) game {
	w, h := termbox.Size()
	snake := newSnake(1, right)
	fx, fy := freeSpot(w/2, h, snake.nodes)
	return game{
		width:  w / 2,
		height: h,
		snake:  snake,
		state:  playing,
		ticker: time.NewTicker(80 * time.Millisecond),
		events: events,
		food:   &food{fx, fy},
	}
}

func main() {
	err := termbox.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer termbox.Close()

	events := make(chan termbox.Event)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()
	game := newGame(events)
	game.loop()
}

// puts on center
func cputs(s string) {
	sx, sy := termbox.Size()
	x := sx/2 - len(s)/2 - 1
	y := sy/2 - 1
	puts(s, x, y)
}

func puts(s string, x, y int) {
	for i, ch := range s {
		termbox.SetCell(x+i, y, ch, termbox.Attribute(4), termbox.ColorDefault)
	}
	termbox.Flush()
}

func freeSpot(width, height int, nodes []*node) (int, int) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	x := r.Intn(width-2) + 1
	y := r.Intn(height-2) + 1
	for _, n := range nodes {
		if (n.x == x) && (n.y == y) {
			return freeSpot(width, height, nodes)
		}
	}
	return x, y
}

package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

var (
	addrFlag = flag.String("addr", ":5555", "server address:port")
)

const (
	kB = 1
	T  = 3
)

type Spin struct {
	Val float64 // -1 or +1
}

type Grid struct {
	M [][]Spin
	N int     // number of rows and columns (they are both equal to the same value: N)
	J float64 // coupling
}

func NewGrid(n int, j float64) *Grid {
	G := &Grid{}
	G.M = make([][]Spin, n)
	for i, _ := range G.M {
		G.M[i] = make([]Spin, n)
	}
	G.N = n
	G.J = j
	return G
}

func (g *Grid) Init() {
	for i := range g.M {
		for j := range g.M[i] {
			s := Spin{}
			rand := rand.Float64()
			if rand < 0.5 {
				s.Val = -1
			} else {
				s.Val = +1
			}
			g.M[i][j] = s
		}
	}
}

func (g *Grid) PickRandomSpin() (int, int) {
	i := rand.Intn(g.N)
	j := rand.Intn(g.N)
	return i, j
}

func (g *Grid) FlipSpin(i, j int) {
	g.M[i][j].Val *= -1
}

// There are 4 nearest neighbours and for each of them, we store the two grid coordinates i (row number) and j (column number)
type NearestNeighbours struct {
	top, right, bottom, left [2]int
}

func (n *NearestNeighbours) Array() [4][2]int {
	return [4][2]int{n.top, n.right, n.bottom, n.left}
}

func (g *Grid) FindNearestNeighbours(i, j int) NearestNeighbours {
	NNs := NearestNeighbours{}
	NNs.top = [2]int{i - 1, j}
	NNs.right = [2]int{i, j + 1}
	NNs.bottom = [2]int{i + 1, j}
	NNs.left = [2]int{i, j - 1}
	if i == 0 {
		NNs.top[0] = g.N - 1
	}
	if i == g.N-1 {
		NNs.bottom[0] = 0
	}
	if j == 0 {
		NNs.left[1] = g.N - 1
	}
	if j == g.N-1 {
		NNs.right[1] = 0
	}
	return NNs
}

func (g *Grid) SpinEnergy(i, j int) float64 {
	NNs := g.FindNearestNeighbours(i, j)
	NNsArr := NNs.Array()
	var energy float64
	for _, nn := range NNsArr {
		iNN := nn[0]
		jNN := nn[1]
		energy += -1 * g.J * g.M[i][j].Val * g.M[iNN][jNN].Val
	}
	return energy
}

type Points struct {
	N int
	X []float64
	Y []float64
}

func NewPoints(grid Grid, spinVal float64) *Points {
	points := &Points{}
	for i := range grid.M {
		for j := range grid.M[i] {
			s := grid.M[i][j]
			if s.Val == spinVal {
				points.N += 1
				points.X = append(points.X, float64(i))
				points.Y = append(points.Y, float64(j))
			}
		}
	}
	return points
}

func (p *Points) Len() int {
	return p.N
}

func (p *Points) XY(i int) (x, y float64) {
	return p.X[i], p.Y[i]
}

func Plot(grid Grid) {
	pointsUp := NewPoints(grid, +1)
	pointsDown := NewPoints(grid, -1)

	scaUp, _ := plotter.NewScatter(pointsUp)
	scaDown, _ := plotter.NewScatter(pointsDown)

	scaUp.GlyphStyle.Color = color.RGBA{255, 0, 0, 255}
	scaDown.GlyphStyle.Color = color.RGBA{0, 0, 255, 255}
	scaUp.GlyphStyle.Radius = vg.Points(3.5)
	scaDown.GlyphStyle.Radius = vg.Points(3.5)
	scaUp.GlyphStyle.Shape = draw.BoxGlyph{}
	scaDown.GlyphStyle.Shape = draw.BoxGlyph{}

	p, _ := plot.New()
	p.Add(scaUp, scaDown)

	s := renderSVG(p)
	datac <- Plots{Plot: s}
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UTC().UnixNano())

	go webServer(addrFlag)

	grid := NewGrid(80, 1)
	grid.Init()
	Plot(*grid)
	for k := 0; k < 10000000; k++ {
		//time.Sleep(1 * time.Millisecond)
		if k%500000 == 0 {
			fmt.Println("k=", k)
		}
		i, j := grid.PickRandomSpin()
		eBef := grid.SpinEnergy(i, j)
		grid.FlipSpin(i, j)
		eAft := grid.SpinEnergy(i, j)
		deltaE := eAft - eBef
		if deltaE > 0 {
			prob := math.Exp(-deltaE / (kB * T))
			rnd := rand.Float64()
			if prob < rnd { // undo spin flip (don't accept change)
				grid.FlipSpin(i, j)
			}
		}
		if k%40000 == 0 {
			Plot(*grid)
		}
	}
}

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
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

var (
	addrFlag = flag.String("addr", ":5555", "server address:port")
)

const (
	kB = 1
)

type Spin struct {
	Val float64 // -1 or +1
}

type Grid struct {
	M [][]Spin
	N int     // number of rows and columns (they are both equal to the same value: N)
	J float64 // coupling
	T float64 // temperature
}

func NewGrid(n int, j, t float64) *Grid {
	G := &Grid{}
	G.M = make([][]Spin, n)
	for i, _ := range G.M {
		G.M[i] = make([]Spin, n)
	}
	G.N = n
	G.J = j
	G.T = t
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

func (g *Grid) Energy() float64 {
	var ene float64
	for i := range g.M {
		for j := range g.M[i] {
			ene += g.SpinEnergy(i, j)
		}
	}
	return ene / float64(g.N*g.N) / 2.
}

func (g *Grid) Mag() float64 {
	var mag float64
	for i := range g.M {
		for j := range g.M[i] {
			mag += g.M[i][j].Val
		}
	}
	return mag / float64(g.N*g.N)
}

func (g *Grid) Move() {
	i, j := g.PickRandomSpin()
	eBef := g.SpinEnergy(i, j)
	g.FlipSpin(i, j)
	eAft := g.SpinEnergy(i, j)
	deltaE := eAft - eBef
	if deltaE > 0 {
		prob := math.Exp(-deltaE / (kB * g.T))
		rnd := rand.Float64()
		if prob < rnd { // undo spin flip (don't accept change)
			g.FlipSpin(i, j)
		}
	}
}

func (g *Grid) Evolve(nSteps int, plot bool) {
	for k := 0; k < nSteps; k++ {
		//time.Sleep(1 * time.Millisecond)
		// 		if k%500000 == 0 {
		// 			fmt.Println("k=", k)
		// 		}
		g.Move()
		if plot && k%40000 == 0 {
			Plot(g, nil, nil, nil)
		}
	}
}

type Points struct {
	N int
	X []float64
	Y []float64
}

func NewPoints(grid *Grid, spinVal float64) *Points {
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

func Plot(grid *Grid, T []float64, E []float64, Mag []float64) {
	sGrid := ""
	if grid != nil {
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

		sGrid = renderSVG(p)
	}
	if T != nil && E != nil {
		pts := make(plotter.XYs, len(T))
		for i := range T {
			pts[i].X = T[i]
			pts[i].Y = E[i]
		}
		p, err := plot.New()
		if err != nil {
			panic(err)
		}

		p.Title.Text = "Energy vs temperature"
		p.X.Label.Text = "X"
		p.Y.Label.Text = "Y"
		err = plotutil.AddLinePoints(p, pts)
		if err != nil {
			panic(err)
		}
		// Save the plot to a PNG file.
		if err := p.Save(6*vg.Inch, 3*vg.Inch, "energyVstemp.png"); err != nil {
			panic(err)
		}
	}
	if T != nil && Mag != nil {
		pts := make(plotter.XYs, len(T))
		for i := range T {
			pts[i].X = T[i]
			pts[i].Y = Mag[i]
		}
		p, err := plot.New()
		if err != nil {
			panic(err)
		}

		p.Title.Text = "Magnetization vs temperature"
		p.X.Label.Text = "X"
		p.Y.Label.Text = "Y"
		err = plotutil.AddLinePoints(p, pts)
		if err != nil {
			panic(err)
		}
		// Save the plot to a PNG file.
		if err := p.Save(6*vg.Inch, 3*vg.Inch, "magVstemp.png"); err != nil {
			panic(err)
		}
	}
	datac <- Plots{Plot: sGrid}
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UTC().UnixNano())

	go webServer(addrFlag)

	N := 16

	nT := 50
	nThermal := math.Pow(2, 10) * float64(N*N)
	nMC := math.Pow(2, 10)

	temps := make([]float64, nT)
	for i := range temps {
		temps[i] = 1 + float64(i)*(4-1)/float64(len(temps))
	}
	//temps := []float64{0.6, 0.8, 1, 1.2, 1.4, 1.6, 1.8, 2., 2.2, 2.4, 2.6, 2.8, 3, 3.2, 3.4}
	//temps := []float64{2, 2.269, 3}
	var energies []float64
	var mags []float64
	energies = make([]float64, len(temps))
	mags = make([]float64, len(temps))

	for iT, temp := range temps {
		fmt.Println("Temperature =", temp)
		grid := NewGrid(N, 1, temp)
		grid.Init()
		grid.Evolve(int(nThermal), true)
		var ene float64
		var mag float64
		var ene2 float64
		var mag2 float64
		for k := 0; k < int(nMC); k++ {
			for kk := 0; kk < N*N; kk++ {
				grid.Move()
			}
			eneloc := grid.Energy()
			magloc := grid.Mag()
			ene += eneloc
			mag += magloc
			ene2 += eneloc * eneloc
			mag2 += magloc * magloc
			// 			fmt.Println("ene, mag=", ene, mag)

		}
		energies[iT] = 1 / (nMC * float64(N*N)) * ene
		mags[iT] = 1 / (nMC * float64(N*N)) * math.Abs(mag)
	}
	Plot(nil, temps, energies, mags)
}

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/net/websocket"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgsvg"

	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
)

const (
	kB = 1   // 1.38e-23 // kg.s^{-2}.K^{-1}
	T  = 0.5 // Kelvin
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

func (g *Grid) Plot() {
	h2d := hbook.NewH2D(g.N, 0, float64(g.N), g.N, 0, float64(g.N))
	for i := range g.M {
		for j := range g.M[i] {
			s := g.M[i][j]
			if s.Val == 1 {
				h2d.Fill(float64(i), float64(j), 1)
			} else {
				h2d.Fill(float64(i), float64(j), 0)
			}
		}
	}
	plotH2D(h2d)
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

func main() {
	flag.Parse()
	rand.Seed(time.Now().UTC().UnixNano())

	go webServer()

	grid := NewGrid(10, 1)
	grid.Init()
	grid.Plot()
	for k := 0; k < 100000; k++ {
		time.Sleep(10 * time.Millisecond)
		i, j := grid.PickRandomSpin()
		fmt.Println("\ni, j = ", i, j)
		eBef := grid.SpinEnergy(i, j)
		grid.FlipSpin(i, j)
		grid.Plot()
		eAft := grid.SpinEnergy(i, j)
		fmt.Println("eBef, eAft = ", eBef, eAft)
		deltaE := eAft - eBef
		if deltaE > 0 {
			prob := math.Exp(-deltaE / (kB * T))
			fmt.Println("prob=", prob)
			rnd := rand.Float64()
			if prob < rnd { // undo spin flip
				grid.FlipSpin(i, j)
				eAftUndo := grid.SpinEnergy(i, j)
				fmt.Println("eAftUndo = ", eAftUndo)
			}
		}
		grid.Plot()
	}

	time.Sleep(3000 * time.Millisecond)
}

var (
	addrFlag = flag.String("addr", ":5555", "server address:port")
	datac    = make(chan plots)
)

type plots struct {
	H2D string `json:"h2d"`
}

func webServer() {
	http.HandleFunc("/", plotHandle)
	http.Handle("/data", websocket.Handler(dataHandler))
	err := http.ListenAndServe(*addrFlag, nil)
	if err != nil {
		panic(err)
	}
}

func plotH2D(h2d *hbook.H2D) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"
	p.X.Tick.Marker = &hplot.FreqTicks{N: 11, Freq: 2}
	p.Y.Tick.Marker = &hplot.FreqTicks{N: 11, Freq: 2}
	p.X.Min = h2d.XMin()
	p.Y.Min = h2d.YMin()
	p.X.Max = h2d.XMax()
	p.Y.Max = h2d.YMax()
	p.Add(hplot.NewH2D(h2d, nil))

	s := renderSVG(p)
	datac <- plots{s}
}

func renderSVG(p *plot.Plot) string {
	size := 20 * vg.Centimeter
	canvas := vgsvg.New(size, size)
	p.Draw(draw.New(canvas))
	out := new(bytes.Buffer)
	_, err := canvas.WriteTo(out)
	if err != nil {
		panic(err)
	}
	return string(out.Bytes())
}

func plotHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, page)
}

func dataHandler(ws *websocket.Conn) {
	for data := range datac {
		err := websocket.JSON.Send(ws, data)
		if err != nil {
			log.Printf("error sending data: %v\n", err)
			return
		}
	}
}

const page = `
<html>
	<head>
		<title>Plotting stuff with gonum/plot</title>
		<script type="text/javascript">
		var sock = null;
		var h2dplot = "";

		function update() {
			var p3 = document.getElementById("my-h2d-plot");
			p3.innerHTML = h2dplot;
		};

		window.onload = function() {
			sock = new WebSocket("ws://"+location.host+"/data");

			sock.onmessage = function(event) {
				var data = JSON.parse(event.data);
				//console.log("data: "+JSON.stringify(data));
				h2dplot = data.h2d;
				update();
			};
		};

		</script>

		<style>
		.my-plot-style {
			width: 400px;
			height: 200px;
			font-size: 14px;
			line-height: 1.2em;
		}
		</style>
	</head>

	<body>
		<div id="header">
			<h2>My plot</h2>
		</div>

		<div id="content">
			<div id="my-h2d-plot" class="my-plot-style"></div>
		</div>
	</body>
</html>
`

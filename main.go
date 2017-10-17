package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
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

type Spin struct {
	Val int // -1 or +1
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
			}
		}
	}
	plotH2D(h2d)
}

func (g *Grid) FlipRandomSpin() {
	i := rand.Intn(g.N)
	j := rand.Intn(g.N)
	g.M[i][j].Val *= -1
}

/*
func (g *Grid) Energy() {
	for i := range g.M {
		for j := range g.M[i] {

		}
	}
}
*/

func main() {
	flag.Parse()

	go webServer()

	grid := NewGrid(10, 1)
	grid.Init()
	for {
		time.Sleep(100 * time.Millisecond)
		grid.Plot()
		grid.FlipRandomSpin()
	}
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

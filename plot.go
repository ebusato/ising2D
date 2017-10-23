// This file contains helper functions to perform web-based plotting

package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgsvg"
)

var (
	datac = make(chan Plots)
)

type Plots struct {
	Plot string `json:"plot"`
}

func webServer(addrFlag *string) {
	http.HandleFunc("/", plotHandle)
	http.Handle("/data", websocket.Handler(dataHandler))
	err := http.ListenAndServe(*addrFlag, nil)
	if err != nil {
		panic(err)
	}
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
		var plot = "";

		function update() {
			var p1 = document.getElementById("my-plot");
			p1.innerHTML = plot;
		};

		window.onload = function() {
			sock = new WebSocket("ws://"+location.host+"/data");

			sock.onmessage = function(event) {
				var data = JSON.parse(event.data);
				//console.log("data: "+JSON.stringify(data));
				plot = data.plot;
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
			<h2>Simulation output</h2>
		</div>

		<div id="content">
			<div id="my-plot" class="my-plot-style"></div>
		</div>
	</body>
</html>
`

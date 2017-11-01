package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/cmplx"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type config struct {
	xmin, ymin, xmax, ymax float64
	width, height          int
	workers                int
}

func (c *config) setPositionStr(posStr string) error {
	p := strings.Split(posStr, ",")
	if len(p) != 4 {
		return fmt.Errorf("Invalid position string : %s", posStr)
	}

	var err error

	c.xmin, err = strconv.ParseFloat(p[0], 64)
	if err != nil {
		return err
	}

	c.ymin, err = strconv.ParseFloat(p[1], 64)
	if err != nil {
		return err
	}

	c.xmax, err = strconv.ParseFloat(p[2], 64)
	if err != nil {
		return err
	}

	c.ymax, err = strconv.ParseFloat(p[3], 64)
	if err != nil {
		return err
	}

	return nil
}

func getConfig(r *http.Request) (*config, error) {
	c := &config{-2, -2, 2, 2, 4096, 4096, 8}

	posStr := r.FormValue("pos")
	if posStr != "" {
		err := c.setPositionStr(posStr)
		if err != nil {
			return nil, err
		}
	}

	widthStr := r.FormValue("width")
	if widthStr != "" {
		twidth, err := strconv.ParseInt(widthStr, 10, 32)
		if err != nil {
			return nil, err
		}
		c.width = int(twidth)
	}
	heightStr := r.FormValue("height")
	if heightStr != "" {
		theight, err := strconv.ParseInt(heightStr, 10, 32)
		if err != nil {
			return nil, err
		}
		c.height = int(theight)
	}

	workersStr := r.FormValue("workers")
	if workersStr != "" {
		tworkers, err := strconv.ParseInt(workersStr, 10, 32)
		if err != nil {
			return nil, err
		}
		c.workers = int(tworkers)
	}

	return c, nil
}

func renderMandelbrot(c *config) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	q := make(chan int, 1000)

	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for py := range q {
				for px := 0; px < c.width; px++ {
					x := float64(px)/float64(c.width)*(c.xmax-c.xmin) + c.xmin
					y := float64(py)/float64(c.height)*(c.ymax-c.ymin) + c.ymin
					z := complex(x, y)
					// Image point (px, py) represents complex value z.
					img.Set(px, py, mandelbrot(z))
				}
			}
		}()
	}

	for py := 0; py < c.height; py++ {
		q <- py
	}
	close(q)

	wg.Wait()

	return img, nil
}

func mandelHandler(w http.ResponseWriter, r *http.Request) {
	c, err := getConfig(r)
	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		return
	}

	ts := time.Now()

	// traceFile, err := os.Create("mandel.trace")
	// if err != nil {
	// 	log.Println(err)
	// } else {
	// 	trace.Start(traceFile)
	// }

	img, err := renderMandelbrot(c)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	// trace.Stop()

	fmt.Println(time.Now().Sub(ts))

	err = png.Encode(w, img)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}
}

func mandelbrot(z complex128) color.Color {
	const iterations = 200
	const contrast = 30

	var v complex128
	for n := uint8(0); n < iterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			return color.Gray{255 - contrast*n}
		}
	}
	return color.Black
}

func main() {
	http.HandleFunc("/mandel", mandelHandler)

	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"os"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
)

var (
	//go:embed map-item-test/dQw4w9WgXcQ.jpg
	bytes1 []byte // Test non-square.
	//go:embed map-item-test/sus.png
	bytes2           []byte // Test transparency and 128 * 128.
	pixels1, pixels2 [][]color.RGBA
)

func images() {
	type Ctx struct {
		Name    string
		Bytes   []byte
		Done    func([][]color.RGBA)
		Decoder func(r io.Reader) (image.Image, error)
		Config  func(r io.Reader) (image.Config, error)
	}
	f := func(ctx Ctx) {
		var (
			log    = logrus.StandardLogger().WithField("image context", ctx.Name)
			buffer = bytes.NewBuffer(ctx.Bytes)

			imageConfig image.Config
			img         image.Image
			err         error
		)

		log.Info("Decoding config")
		if imageConfig, err = ctx.Config(buffer); err != nil {
			panic(err)
		}

		log.Info("Decoding pixels")
		if img, err = ctx.Decoder(buffer); err != nil {
			panic(err)
		}
		height := imageConfig.Height
		pixels := make([][]color.RGBA, height)
		for x := 0; x < height; x++ {
			pixels[x] = make([]color.RGBA, height)
			for y := 0; y < imageConfig.Width; y++ {
				r, g, b, a := img.At(y, x).RGBA()
				pixels[x][y] = color.RGBA{
					R: uint8(r),
					G: uint8(g),
					B: uint8(b),
					A: uint8(a),
				}
			}
		}

		ctx.Done(pixels)
	}

	go f(Ctx{"Test non-square", bytes1, func(pixels [][]color.RGBA) {
		pixels1 = pixels
	}, jpeg.Decode, jpeg.DecodeConfig})
	go f(Ctx{"Test transparency and 128 * 128", bytes2, func(pixels [][]color.RGBA) {
		pixels2 = pixels
	}, png.Decode, png.DecodeConfig})
}

func main() {
	log := logrus.StandardLogger()
	log.Formatter = &logrus.TextFormatter{ForceColors: true}
	log.Level = logrus.DebugLevel

	images()

	chat.Global.Subscribe(chat.StdoutSubscriber{})

	config, err := readConfig()
	if err != nil {
		log.Fatalln(err)
	}

	srv := server.New(&config, log)
	srv.CloseOnProgramEnd()
	if err := srv.Start(); err != nil {
		log.Fatalln(err)
	}

	var d1, d2 *world.ViewableMapData
	data1 := func() *world.ViewableMapData {
		if d1 == nil {
			d1 = world.NewMapData()
			d1.ChangePixels()
		}

		return d1
	}
	data2 := func() *world.ViewableMapData {
		if d1 == nil {
			d1 = world.NewMapData()
		}

		return d1
	}

	for srv.Accept(func(p *player.Player) {
		for l := 0; l < 3; l++ {
			for _, i := range []world.Item{
				item.FilledMap{},
				item.OceanMap{},
				item.WoodlandExplorerMap{},
				item.TreasureMap{},
			} {
				switch l {
				case 1:
					l.ViewableMapData = data1()
				}

				if _, err := p.Inventory().AddItem(item.NewStack(i, 1)); err != nil {
					panic(err)
				}
			}
		}
	}) {
	}
}

// readConfig reads the configuration from the config.toml file, or creates the file if it does not yet exist.
func readConfig() (server.Config, error) {
	c := server.DefaultConfig()
	if _, err := os.Stat("config.toml"); os.IsNotExist(err) {
		data, err := toml.Marshal(c)
		if err != nil {
			return c, fmt.Errorf("failed encoding default config: %v", err)
		}
		if err := ioutil.WriteFile("config.toml", data, 0644); err != nil {
			return c, fmt.Errorf("failed creating config: %v", err)
		}
		return c, nil
	}
	data, err := ioutil.ReadFile("config.toml")
	if err != nil {
		return c, fmt.Errorf("error reading config: %v", err)
	}
	if err := toml.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("error decoding config: %v", err)
	}
	return c, nil
}

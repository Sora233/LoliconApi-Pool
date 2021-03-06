# LoliconApi Pool

an image pool for [LoliconApi](https://api.lolicon.app) in Golang

## Feature

- [x] Always request 10 image, cache unused ones.
- [x] Support Keyword

## Usage

```go
package main

import (
	"bytes"
	"fmt"
	loliconApiPool "github.com/Sora233/LoliconApi-Pool"
	"github.com/sirupsen/logrus"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
)

func main() {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	pool, err := loliconApiPool.NewLoliconPool(&loliconApiPool.Config{
		ApiKey:   "", // use your api key here
		CacheMin: 0,
		CacheMax: 1,
		Persist:  loliconApiPool.NewNilPersist(),
	})
	if err != nil {
		panic(err)
	}
	// use the pool
	img, err := pool.Get(
		loliconApiPool.NumOption(1),
		loliconApiPool.R18Option(loliconApiPool.R18Off),
	)
	if err != nil {
		panic(err)
	}
	for _, i := range img {
		b, err := i.Content()
		if err != nil {
			panic(err)
		}
		if len(b) == 0 {
			continue
		}
		_, cfg, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil {
			logrus.Errorf("unknown format")
			continue
		}
		filename := fmt.Sprintf("%v.%v", i.Pid, cfg)
		ioutil.WriteFile(filename, b, 0644)
		logrus.WithField("filename", filename).Info("saved")
	}
}

```
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
	"github.com/Sora233/LoliconApiPool"
	"image"
	"io/ioutil"
)

func main() {
	pool, err := loliconApiPool.NewLoliconPool(&loliconApiPool.Config{
		ApiKey:   "your api key",
		CacheMin: 5,
		CacheMax: 20,
		Persist:  loliconApiPool.NewNilPersist(),
	})
	if err != nil {
		panic(err)
	}
	// use the pool
	img, err := pool.Get(loliconApiPool.R18Option(loliconApiPool.R18Off))
	if err != nil {
		panic(err)
	}
	for _, i := range img {
		b, err := i.Content()
		if err != nil {
			panic(err)
		}
		_, suf, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(fmt.Sprintf("%v.%v", i.Pid, suf), b, 0644)
	}
}
```
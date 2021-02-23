package loliconApiPool

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

var logger = logrus.NewEntry(logrus.StandardLogger())

type Config struct {
	ApiKey   string
	CacheMin int
	CacheMax int
	Persist  Persist
}

type LoliconPool struct {
	config *Config
	cache  map[R18Type]*list.List
	cond   *sync.Cond

	Persist
}

type Option map[string]interface{}

type OptionFunc func(option Option) Option

func KeywordOption(keyword string) OptionFunc {
	return func(option Option) Option {
		option["keyword"] = keyword
		return option
	}
}

func NumOption(num int) OptionFunc {
	return func(option Option) Option {
		option["num"] = num
		return option
	}
}

func R18Option(r18Type R18Type) OptionFunc {
	return func(option Option) Option {
		option["r18"] = r18Type
		return option
	}
}

// caller must hold the lock
func (pool *LoliconPool) fillCacheFromRemote(r18 R18Type) error {
	logger.WithField("r18", r18.String()).Debug("fetch from remote")
	resp, err := LoliconAppSetu(pool.config.ApiKey, r18, "", 10)
	if err != nil {
		return err
	}
	logger.WithField("Quota", resp.Quota).
		WithField("QuotaMinTTL", resp.QuotaMinTTL).
		WithField("Msg", resp.Msg).
		WithField("Code", resp.Code).
		Debug("LoliconPool response")
	if resp.Code != 0 {
		return fmt.Errorf("response code %v: %v", resp.Code, resp.Msg)
	}
	for _, s := range resp.Data {
		pool.cache[r18].PushFront(s)
	}
	return nil
}

func (pool *LoliconPool) background() {
	go func() {
		for range time.Tick(time.Second * 30) {
			pool.storeIntoPersist()
		}
	}()
	for {
		var result = true
		pool.cond.L.Lock()
		for {
			var checkResult = false
			for _, v := range pool.cache {
				if v.Len() < pool.config.CacheMin {
					checkResult = true
				}
			}
			if checkResult {
				break
			}
			pool.cond.Wait()
		}
		for r18, l := range pool.cache {
			if l.Len() < pool.config.CacheMin {
				for l.Len() < pool.config.CacheMax {
					if err := pool.fillCacheFromRemote(r18); err != nil {
						logger.WithField("from", "background").Errorf("fill cache from remote failed %v", err)
						result = false
						break
					}
				}
			}
		}
		pool.cond.L.Unlock()
		if !result {
			time.Sleep(time.Minute)
		}
	}
}

func (pool *LoliconPool) loadFromPersist() {
	pool.cond.L.Lock()
	defer pool.cond.L.Unlock()
	for r18, l := range pool.cache {
		img, err := pool.Load(r18)
		if err != nil {
			logger.WithField("r18", r18.String()).
				Errorf("load from persist failed %v", err)
			continue
		}
		for _, i := range img {
			l.PushBack(i)
		}
		logger.WithField("r18", r18.String()).
			WithField("image_count", l.Len()).
			Debug("load from persist success")
	}
}

func (pool *LoliconPool) storeIntoPersist() {
	pool.cond.L.Lock()
	defer pool.cond.L.Unlock()
	for r18, l := range pool.cache {
		var img []*Setu
		root := l.Front()
		if root == nil {
			continue
		}
		for {
			img = append(img, root.Value.(*Setu))
			if root == l.Back() {
				break
			}
			root = root.Next()
			if root == nil {
				break
			}
		}
		if err := pool.Store(r18, img); err != nil {
			logger.WithField("r18", r18.String()).
				WithField("image_count", l.Len()).
				Errorf("store into persist failed %v", err)
			continue
		}
		logger.WithField("r18", r18.String()).
			WithField("image_count", l.Len()).
			Debugf("store into persist success")
	}
}

func (pool *LoliconPool) getCache(r18 R18Type, num int) (result []*Setu, err error) {
	pool.cond.L.Lock()
	defer pool.cond.L.Unlock()
	for i := 0; i < num; i++ {
		if pool.cache[r18].Len() == 0 {
			err = pool.fillCacheFromRemote(r18)
			if err != nil {
				logger.WithField("from", "getCache").Errorf("fill cache from remote failed %v", err)
				break
			}
		}
		result = append(result, pool.cache[r18].Remove(pool.cache[r18].Front()).(*Setu))
	}
	pool.cond.Signal()
	return
}

func (pool *LoliconPool) Get(options ...OptionFunc) ([]*Setu, error) {
	option := make(Option)
	for _, optionFunc := range options {
		optionFunc(option)
	}

	var (
		r18     R18Type
		keyword string
		num     int
	)
	for k, v := range option {
		switch k {
		case "keyword":
			_v, ok := v.(string)
			if ok {
				keyword = _v
			}
		case "num":
			_v, ok := v.(int)
			if ok {
				num = _v
			}
		case "r18":
			_v, ok := v.(R18Type)
			if ok {
				r18 = _v
			}
		}
	}
	if keyword != "" {
		logger.Debugf("request remote image")
		resp, err := LoliconAppSetu(pool.config.ApiKey, r18, keyword, num)
		if err != nil {
			return nil, err
		}
		logger.WithField("image num", len(resp.Data)).
			WithField("quota", resp.Quota).
			WithField("quota_min_ttl", resp.QuotaMinTTL).
			Debugf("request done")
		if resp.Code != 0 {
			return nil, fmt.Errorf("response code %v: %v", resp.Code, resp.Msg)
		}
		var result []*Setu
		for _, img := range resp.Data {
			result = append(result, img)
		}
		return result, nil
	}
	return pool.getCache(r18, num)
}

func NewLoliconPool(config *Config) (*LoliconPool, error) {
	if config.ApiKey == "" {
		return nil, errors.New("empty api key")
	}
	if config.CacheMin <= 0 {
		config.CacheMin = 0
	}
	if config.CacheMax <= 0 {
		config.CacheMax = 50
	}
	if config.CacheMin > config.CacheMax {
		config.CacheMin = config.CacheMax
	}
	if config.Persist == nil {
		config.Persist = NewNilPersist()
	}
	pool := &LoliconPool{
		config:  config,
		cache:   make(map[R18Type]*list.List),
		cond:    sync.NewCond(&sync.Mutex{}),
		Persist: config.Persist,
	}
	pool.cache[R18Off] = list.New()
	pool.cache[R18On] = list.New()
	pool.loadFromPersist()
	go pool.background()
	return pool, nil
}

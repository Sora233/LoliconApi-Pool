package loliconApiPool

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const Host = "https://api.lolicon.app/setu"

var c = &http.Client{
	Timeout: time.Second * 60,
}

type R18Type int

const (
	R18Off R18Type = iota
	R18On
	//R18Mix
)

func (r R18Type) String() string {
	switch r {
	case R18Off:
		return "R18Off"
	case R18On:
		return "R18On"
	//case R18Mix:
	//	return "R18Mix"
	default:
		return "Unknown"
	}
}

type Request struct {
	Apikey   string `json:"apikey"`
	R18      int    `json:"r18"`
	Keyword  string `json:"keyword"`
	Num      int    `json:"num"`
	Proxy    string `json:"proxy"`
	Size1200 bool   `json:"size1200"`
}

type Setu struct {
	Pid    int      `json:"pid"`
	P      int      `json:"p"`
	Uid    int      `json:"uid"`
	Title  string   `json:"title"`
	Author string   `json:"author"`
	Url    string   `json:"url"`
	R18    bool     `json:"r18"`
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Tags   []string `json:"tags"`
}

func (s *Setu) Content() ([]byte, error) {
	if s == nil {
		return nil, errors.New("<nil>")
	}
	resp, err := c.Get(s.Url)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

type Response struct {
	Code        int     `json:"code"`
	Msg         string  `json:"msg"`
	Quota       int     `json:"quota"`
	QuotaMinTTL int     `json:"quota_min_ttl"`
	Count       int     `json:"count"`
	Data        []*Setu `json:"data"`
}

func LoliconAppSetu(apikey string, R18 R18Type, keyword string, num int) (*Response, error) {
	params, err := ToParams(&Request{
		Apikey:   apikey,
		R18:      int(R18),
		Keyword:  keyword,
		Num:      num,
		Proxy:    "i.pixiv.cat",
		Size1200: true,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, Host, nil)
	if err != nil {
		return nil, err
	}
	q := make(url.Values)
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	apiResp := new(Response)
	bodyb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bodyb, &apiResp)
	if err != nil {
		return nil, err
	}
	return apiResp, nil
}

func ToParams(get interface{}) (map[string]string, error) {
	params := make(map[string]string)

	rg := reflect.ValueOf(get)
	if rg.Type().Kind() == reflect.Ptr {
		rg = rg.Elem()
	}
	if rg.Type().Kind() != reflect.Struct {
		return nil, errors.New("can only convert struct type")
	}
	for i := 0; ; i++ {
		if i >= rg.Type().NumField() {
			break
		}
		field := rg.Type().Field(i)
		fillname, found := field.Tag.Lookup("json")
		if !found {
			fillname = toCamel(field.Name)
		} else {
			if pos := strings.Index(fillname, ","); pos != -1 {
				fillname = fillname[:pos]
			}
		}
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			params[fillname] = strconv.FormatInt(rg.Field(i).Int(), 10)
		case reflect.String:
			params[fillname] = rg.Field(i).String()
		case reflect.Bool:
			params[fillname] = strconv.FormatBool(rg.Field(i).Bool())
		default:
			return nil, fmt.Errorf("not support type %v", field.Type.Kind().String())
		}

	}
	return params, nil
}

func toCamel(name string) string {
	if len(name) == 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.WriteString(strings.ToLower(name[:1]))
	for _, c := range name[1:] {
		if c >= 'A' && c <= 'Z' {
			sb.WriteRune('_')
			sb.WriteRune(c - 'A' + 'a')
		} else {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

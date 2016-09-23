package server

import (
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/kubernetes/staging/src/k8s.io/client-go/1.4/pkg/util/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type MetaResponse struct {
	Response `json:"response"`
}

type Response struct {
	Count    int       `json:"count"`
	Messages []Message `json:"items"`
}

type Message struct {
	Body        string        `json:"body"`
	UserID      int           `json:"user_id"`
	Attachments *[]Attachment `json:"attachments"`
	ReadState   int           `json:"read_state"` // 1 - read, 0 - unread
}

type Attachment struct {
	Type  string `json:"type"`
	Audio Audio  `json:"audio"`
}

type Audio struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Url    string `json:"url"`
}

const (
	vk_api string = `https://api.vk.com/method`
)

func GetNewTracks() {
	c := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(`GET`, vk_api+`/messages.get`, nil)
	if err != nil {
		panic(err)
	}
	q := req.URL.Query()
	q.Add(`access_token`, vk_token)
	q.Add(`out`, `0`)
	q.Add(`v`, `5.53`)
	req.URL.RawQuery = q.Encode()

	var r MetaResponse
	for {
		time.Sleep(10 * time.Second)
		resp, err := c.Do(req)
		if err != nil {
			panic(err)
		}

		d := json.NewDecoder(resp.Body)
		err = d.Decode(&r)
		if err != nil {
			panic(err)
		}

		for _, m := range r.Messages {
			if m.ReadState != 1 && m.Attachments != nil {
				for _, a := range *m.Attachments {
					if a.Type == `audio` {
						ch <- MyChan{
							UserID: m.UserID,
							Audio:  a.Audio,
						}
					}
				}
			}
		}
	}
}

func DownloadMusic(in <-chan MyChan) {
	c := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(`GET`, vk_api+`/messages.send`, nil)
	if err != nil {
		panic(err)
	}

	q := req.URL.Query()
	q.Add(`access_token`, vk_token)
	q.Add(`peer_id`, `0`)
	q.Add(`message`, ``)
	q.Add(`v`, `5.53`)
	path := conf.MusicDirectory
	os.MkdirAll(path, 0777)

	answer := func(id, msg string) {
		q.Set(`peer_id`, id)
		q.Set(`message`, msg)
		req.URL.RawQuery = q.Encode()
		c.Do(req)
	}

	for {
		mc := <-in
		a := mc.Audio
		id := strconv.Itoa(mc.UserID)

		// Выкачивание трека
		// Иногда ВК не даёт ссылку на трек
		if a.Url == `` {
			msg := `Invalid track: ` + a.Artist + a.Title + antiflood()
			answer(id, msg)
			continue
		}

		resp, err := http.Get(a.Url)
		if err != nil {
			panic(err)
		}

		fn := fmt.Sprintf(`%s - %s.mp3`, a.Artist, a.Title)
		f, err := os.OpenFile(path+fn, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			panic(err)
		}

		p.NotPlayed = append(p.NotPlayed, fn)
		resp.Body.Close()
		f.Close()

		msg := `Your track is accepted!` + antiflood()
		answer(id, msg)
	}
}

func antiflood() string {
	sym := '\u200B'
	text := ""
	for i := 0; i < rand.IntnRange(0, 20); i++ {
		text += string(sym)
	}
	return text
}

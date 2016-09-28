package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	Id       int    `json:"id"`
	Artist   string `json:"artist"`
	Title    string `json:"title"`
	Url      string `json:"url"`
	Duration int    `json:"duration"`
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
		time.Sleep(2 * time.Second)
		resp, err := c.Do(req)
		if err != nil {
			die(err)
		}

		d := json.NewDecoder(resp.Body)
		err = d.Decode(&r)
		if err != nil {
			die(err)
		}

		for _, m := range r.Messages {
			if m.ReadState != 1 {
				fmt.Println(m)
				if m.UserID == conf.AdminID && strings.Contains(m.Body, `/skip`) {
					skip <- 1
				}
				if m.Attachments != nil {
					for _, a := range *m.Attachments {
						fmt.Println(a)
						if a.Type == `audio` {
							ch <- MyChan{
								UserID: m.UserID,
								Audio:  a.Audio,
							} //wow
						} //wow
					} //wow
				} //wow
			} //wow
		} //wow
	} //wow
} //wow

func download(a Audio, path, fn string) {
	resp, err := http.Get(a.Url)
	if err != nil {
		die(err)
	}

	f, err := os.OpenFile(path+fn, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		die(err)
	}
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		die(err)
	}
	resp.Body.Close()
	f.Close()

	t := Track{Name: fn, Duration: time.Duration(a.Duration) * time.Second}
	tracks = append(tracks, t)
	all = append(all, t)
}

func DownloadMusic(in <-chan MyChan) {
	c := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(`GET`, vk_api+`/messages.send`, nil)
	if err != nil {
		die(err)
	}

	q := req.URL.Query()
	q.Add(`access_token`, vk_token)
	q.Add(`peer_id`, `0`)
	q.Add(`message`, `debug`)
	q.Add(`v`, `5.53`)
	path := conf.MusicDirectory
	os.MkdirAll(path, 0777)

	answer := func(id, msg string) error {
		q.Set(`peer_id`, id)
		q.Set(`message`, msg)
		req.URL.RawQuery = q.Encode()
		_, err := c.Do(req)
		return err
	}

	for {
		mc := <-in
		a := mc.Audio
		id := strconv.Itoa(mc.UserID)

		logText := fmt.Sprintf(`id%d кинул "%s - %s"`, mc.UserID, a.Artist, a.Title)
		log.Println(logText)

		// Выкачивание трека
		// Иногда ВК не даёт ссылку на трек
		if a.Url == `` {
			msg := `Invalid track: ` + a.Artist + a.Title + antiflood()
			if err := answer(id, msg); err != nil {
				die(err)
			}
			continue
		}

		msg := `Your track is accepted!` + antiflood()
		if err := answer(id, msg); err != nil {
			die(err)
		}

		fn := fmt.Sprintf(`%d.mp3`, a.Id)
		if _, err := os.Open(fn); os.IsNotExist(err) {
			go download(a, path, fn)
		}

	}
}

func antiflood() string {
	sym := '\u200B'
	text := ""

	for i := 0; i < rand.Intn(20); i++ {
		text += string(sym)
	}
	return text
}

func die(err error) {
	log.Println(err)
	panic(err)
}

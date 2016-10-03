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
	Body        string       `json:"body"`
	UserID      int          `json:"user_id"`
	Attachments []Attachment `json:"attachments"`
	ReadState   int          `json:"read_state"` // 1 - read, 0 - unread
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
	nl            = "\r\n"
)

func getMessages() {
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

	for {
		// АААААА!!!
		var r MetaResponse
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
				if m.UserID == conf.AdminID && strings.Contains(m.Body, `/skip`) {
					skip <- 1
					msgChan <- Msg{m.UserID, `skipped`}
				} else if len(m.Attachments) > 0 {
					for _, a := range m.Attachments {
						if a.Type == `audio` {
							ch <- AudioChan{
								UserID: m.UserID,
								Audio:  a.Audio,
							}
						}
					}
				} else {
					msgChan <- Msg{m.UserID, `wtf?`}
				}
			}
		}
		resp.Body.Close()
	}
}

func download(a Audio, path, fn string) {
	resp, err := http.Get(a.Url)
	defer resp.Body.Close()
	if err != nil {
		die(err)
	}

	f, err := os.OpenFile(path+fn, os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()
	if err != nil {
		die(err)
	}
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		die(err)
	}

	t := Track{Name: fn, Duration: time.Duration(a.Duration) * time.Second}
	tracks = append(tracks, t)
	all = append(all, t)
}

func sendMessage(in <-chan Msg) {
	c := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(`GET`, vk_api+`/messages.send`, nil)
	if err != nil {
		die(err)
	}

	q := req.URL.Query()
	q.Add(`access_token`, vk_token)
	q.Add(`peer_id`, `0`)
	q.Add(`message`, `woops`)
	q.Add(`v`, `5.53`)
	for {
		m := <-in
		q.Set(`peer_id`, strconv.Itoa(m.UserID))
		q.Set(`message`, m.Message+antiflood())
		req.URL.RawQuery = q.Encode()
		resp, err := c.Do(req)
		if err != nil {
			die(err)
		}

		if resp.StatusCode != 200 {
			log.Print(`error in messages.send`, q, nl)
		}
	}

}
func DownloadMusic(in <-chan AudioChan) {
	path := conf.MusicDirectory
	os.MkdirAll(path, 0777)
	for {
		mc := <-in
		a := mc.Audio

		l := fmt.Sprintf(`id%d кинул "%s - %s"`, mc.UserID, a.Artist, a.Title)
		log.Print(l, nl)

		// Выкачивание трека
		// Иногда ВК не даёт ссылку на трек
		if a.Url == `` {
			msg := `Invalid track: ` + a.Artist + a.Title
			msgChan <- Msg{mc.UserID, msg}
			continue
		}

		fn := fmt.Sprintf(`%d.mp3`, a.Id)
		for _, t := range tracks {
			if fn == t.Name {
				msgChan <- Msg{mc.UserID, `Track is already added`}
				continue
			}
		}

		msgChan <- Msg{mc.UserID, `Your track is accepted!`}
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
	log.Print(err, nl)
	panic(err)
}

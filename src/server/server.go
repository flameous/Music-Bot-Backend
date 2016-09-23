package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
)

type Player struct {
	PlayingNow    string
	AlreadyPlayed []string
	NotPlayed     []string
}

type Config struct {
	Token          string `json:"token"`
	MusicDirectory string `json:"music_directory"`
}

type MyChan struct {
	UserID int
	Audio  Audio
}

var (
	conf     Config
	ch       chan MyChan
	p        Player
	vk_token string
)

func Run() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(file, &conf)
	if err != nil {
		panic(err)
	}

	vk_token = conf.Token

	ch = make(chan MyChan, 1)

	p = Player{
		NotPlayed:     []string{},
		AlreadyPlayed: []string{},
		PlayingNow:    ``,
	}

	go GetNewTracks()
	go DownloadMusic(ch)

	http.HandleFunc(`/play`, play)
	http.ListenAndServe(`:8000`, nil)
}

func play(w http.ResponseWriter, r *http.Request) {
	if len(p.NotPlayed) > 0 {
		p.PlayingNow = p.NotPlayed[0]
		p.NotPlayed = p.NotPlayed[1:]
		p.AlreadyPlayed = append(p.AlreadyPlayed, p.PlayingNow)

		cmd := exec.Command(`killall`, `afplay`)
		cmd.Run()

		cmd = exec.Command(`afplay`, conf.MusicDirectory+p.PlayingNow)
		err := cmd.Start()
		if err != nil {
			panic(err)
		}

		w.Write([]byte(p.PlayingNow))
	} else {
		w.Write([]byte(`We have no music, sorry!`))
	}
	//fmt.Println(`------`)
	//fmt.Println(`Now Playing: `, p.PlayingNow)
	//fmt.Println(`Out`, p.AlreadyPlayed)
	//fmt.Println(`Will play`, p.NotPlayed)
	//fmt.Println(`------`)
}

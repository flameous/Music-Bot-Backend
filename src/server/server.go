package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type Config struct {
	Token          string `json:"token"`
	MusicDirectory string `json:"music_directory"`
	AdminID        int    `json:"admin_id"`
}

type AudioChan struct {
	UserID int
	Audio  Audio
}

type Msg struct {
	UserID  int
	Message string
}

var (
	conf     Config
	vk_token string
	tracks   []Track
	all      []Track
	ch       chan AudioChan
	msgChan  chan Msg
	skip     chan int
	lastfm   chan AudioChan
)

type Track struct {
	Name     string
	Duration time.Duration
}

func Run() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(file, &conf)
	if err != nil {
		panic(err)
	}

	tracks = []Track{}
	all = []Track{}
	logfile, err := os.OpenFile(`logs.log`, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logfile)

	vk_token = conf.Token

	ch = make(chan AudioChan, 100)
	msgChan = make(chan Msg, 100)
	skip = make(chan int, 1)
	lastfm = make(chan AudioChan, 100)

	go getMessages()
	go DownloadMusic(ch)
	go sendMessage(msgChan)
	go scrobble()
	serve()
}

func play(track string) {
	var err error

	switch runtime.GOOS {
	case `windows`:
		err = exec.Command(`vlc.exe`, conf.MusicDirectory+track, `--play-and-exit`).Start()
	default:
		err = exec.Command(`afplay`, conf.MusicDirectory+track).Start()
	}
	if err != nil {
		die(err)
	}

}

func kill() {
	var err error

	switch runtime.GOOS {
	case `windows`:
		err = exec.Command(`taskkill`, `/F`, `/IM`, `vlc.exe`).Run()
	default:
		err = exec.Command(`killall`, `afplay`).Run()
	}
	if err != nil {
		die(err)
	}
}

func serve() {
	for {
		if len(tracks) != 0 {
			current := tracks[0]
			play(current.Name)
			log.Print(`Играет трек ` + tracks[0].Name + nl)
			tracks = tracks[1:]

			select {
			case <-skip:
				kill()

			case <-time.After(current.Duration):
			}
		} else if len(all) != 0 {
			log.Print(fmt.Sprintf(`Плейлист играет заново, %d песен`, len(all)), nl)
			tracks = all
		}
		time.Sleep(2 * time.Second)
	}
}

package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

type Config struct {
	Token          string `json:"token"`
	MusicDirectory string `json:"music_directory"`
	AdminID        int    `json:"admin_id"`
}

type MyChan struct {
	UserID int
	Audio  Audio
}

var (
	conf     Config
	ch       chan MyChan
	vk_token string
	tracks   []Track
	all      []Track
	skip     chan int
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

	ch = make(chan MyChan, 100)
	skip = make(chan int, 1)

	go GetNewTracks()
	go DownloadMusic(ch)
	play()
}

func play() {
	for {
		if len(tracks) != 0 {
			current := tracks[0]
			cmd := exec.Command(`afplay`, conf.MusicDirectory+current.Name)
			if err := cmd.Start(); err != nil {
				die(err)
			}
			log.Println(`Играет трек ` + tracks[0].Name)
			tracks = tracks[1:]

			select {
			case <-time.After(current.Duration):
				continue
			case <-skip:
				exec.Command(`killall`, `afplay`).Run()
			}
		} else if len(all) != 0 {
			log.Println(fmt.Sprintf(`Плейлист играет заново, %d песен`, len(all)))
			tracks = all
		}
		time.Sleep(2 * time.Second)
	}
}

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
	Admins         []int  `json:"admins"`
}

var (
	conf     Config
	vk_token string
	tracks   []Track
	all      []Track
	ch       chan Audio
	action   chan string
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

	action = make(chan string)
	go getMessages()
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
		e := fmt.Sprintf(`Cannot skip, maybe tracklist is empty. Error: %s`+nl, err)
		fmt.Print(e)
		log.Print(e)
	}
}

func serve() {
	for {
		var current Track
		if len(tracks) != 0 {
			current = tracks[0]
			play(current.Name)

			l := fmt.Sprintf(`Играет трек %s`+nl, current.Name)
			fmt.Print(l)
			log.Print(l)
			tracks = tracks[1:]
		}

		select {
		case a := <-action:
			switch a {
			case `skip`:
				kill()
			case `repeat`:
				kill()
				l := fmt.Sprintf(`Плейлист играет заново, %d песен`+nl, len(all))
				fmt.Print(l)
				log.Print(l)
				tracks = all
			}
		case <-time.After(current.Duration):
			// playing next track
		}

		time.Sleep(2 * time.Second)
	}
}

func die(err error) {
	log.Print(err, nl)
	panic(err)
}

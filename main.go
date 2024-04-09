package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	NUM_STREAMS = 1000
	RTMP_URL    = "rtmp://192.168.0.104/l"
)

type Stream struct {
	url string
	ctx context.Context
}

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// file, err := os.OpenFile("logfile.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer file.Close()

	// // Set the log output to the file
	// log.SetOutput(file)

	streams := mount(context.Background())
	start(streams)

	<-sigCh
}

func start(streams []Stream) {
	wg := sync.WaitGroup{}

	rand.NewSource(time.Now().UnixNano())

	for _, s := range streams {
		wg.Add(1)
		// sleepTime := 500 + rand.Intn(300)
		// time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		go func(stream Stream) {
			defer wg.Done()

			args := []string{
				"-loglevel", "error",
				"-re",
				"-stream_loop", "-1",
				"-i", "./assets/2048.flv",
				"-c", "copy",
				"-f", "flv",
				stream.url,
			}

			cmd := exec.CommandContext(stream.ctx, "ffmpeg", args...)

			cmd.Stderr = log.Writer()
			cmd.Stdout = log.Writer()

			log.Println("publishing to", stream.url)

			if err := cmd.Run(); err != nil {
				log.Printf("stream=%s, exit=%v", stream.url, err)
			}
		}(s)
	}

	wg.Wait()
}

func mount(ctx context.Context) []Stream {
	streams := make([]Stream, 0)

	for i := 0; i <= NUM_STREAMS; i++ {
		hash := fmt.Sprintf("TESTLOAD%d", i)
		url := fmt.Sprintf("%s/%s", RTMP_URL, hash)

		streams = append(streams, Stream{
			url: url,
			ctx: ctx,
		})
	}

	return streams
}

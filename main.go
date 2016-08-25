package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gosuri/uiprogress"
)

var usage = `
Usage: plex-sync [token@]IP[:PORT]/SECTION [token@]IP[:PORT]/SECTION [[token@]IP[:PORT]/SECTION...]

Example:

	Sync section 1 on a server with the default port
	with section 2 on another server:
	$ plex-sync 10.0.1.2/1 10.0.1.3:32401/2

	Sync three servers:
	$ plex-sync 10.0.1.2/1 10.0.1.3/1 10.0.1.4/1

	Sync with different tokens:
	$ plex-sync xxxxx@10.0.1.2/1 yyyyy@10.0.1.3/1
`

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, usage)
		os.Exit(1)
	}

	precise := os.Getenv("MATCH_TYPE") == "precise"
	dryRun := os.Getenv("DRY_RUN") != ""

	uiprogress.Start()

	args := os.Args[1:]
	servers := make([]*Server, len(args))

	for idx, arg := range args {
		server, err := ServerFromArg(arg)
		if err != nil {
			log.Fatal(err)
		}
		servers[idx] = server
	}

	watched := make(map[string]bool)

	var wg sync.WaitGroup
	wg.Add(len(servers))

	for _, server := range servers {
		go func(server *Server) {
			defer wg.Done()
			err := server.FetchSection()
			if err != nil {
				log.Fatal(err)
			}

			bar := uiprogress.AddBar(len(server.Videos)).AppendCompleted().PrependElapsed()

			for _, video := range server.Videos {
				bar.Incr()

				if precise {
					server.PopulateGUID(&video)
				}

				if video.Watched() {
					watched[video.Key] = true
				}
			}
		}(server)
	}

	wg.Wait()

	wg.Add(len(servers))
	for _, server := range servers {
		go func(server *Server) {
			defer wg.Done()
			bar := uiprogress.AddBar(len(server.Videos)).AppendCompleted().PrependElapsed()
			for _, video := range server.Videos {
				bar.Incr()
				if watched[video.Key] {
					if !video.Watched() {
						if dryRun {
							fmt.Printf("DRY RUN: would set %s to watched on %s\n", video.Title, server.Host)
						} else {
							server.MarkWatched(&video)
						}
					}
				}
			}
		}(server)
	}
	wg.Wait()
	uiprogress.Stop()
}

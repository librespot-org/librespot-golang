package main

import (
	"Spotify"
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"librespot"
	"librespot/core"
	"librespot/utils"
	"os"
	"strings"
)

const kDefaultDeviceName = "librespot"

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("play <track>:                   play specified track by spotify base64 id")
	fmt.Println("track <track>:                  show details on specified track by spotify base64 id")
	fmt.Println("playlists:                      show your playlist")
	fmt.Println("help:                           show this list\n")
}

func main() {
	// Read flags from commandline
	username := flag.String("username", "", "spotify username")
	password := flag.String("password", "", "spotify password")
	blob := flag.String("blob", "blob.bin", "spotify auth blob")
	devicename := flag.String("devicename", kDefaultDeviceName, "name of device")
	flag.Parse()

	// Authenticate
	var session *core.Session
	var err error

	if *username != "" && *password != "" {
		// Authenticate using a regular login and password, and store it in the blob file.
		session, err = librespot.Login(*username, *password, *devicename)

		err := ioutil.WriteFile(*blob, session.ReusableAuthBlob(), 0600)
		if err != nil {
			fmt.Printf("Could not store authentication blob in blob.bin: %s\n", err)
		}
	} else if *blob != "" && *username != "" {
		// Authenticate reusing an existing blob
		blobBytes, err := ioutil.ReadFile(*blob)

		if err != nil {
			fmt.Printf("Unable to read auth blob from %s: %s\n", *blob, err)
			os.Exit(1)
			return
		}

		session, err = librespot.LoginSaved(*username, blobBytes, *devicename)
	} else if os.Getenv("client_secret") != "" {
		// Authenticate using OAuth (untested)
		session, err = librespot.LoginOAuth(*devicename, os.Getenv("client_id"), os.Getenv("client_secret"))
	} else {
		// No valid options, show the helo
		fmt.Println("need to supply a username and password or a blob file path")
		fmt.Println("./microclient --username SPOTIFY_USERNAME [--blob ./path/to/blob]")
		fmt.Println("or")
		fmt.Println("./microclient --username SPOTIFY_USERNAME --password SPOTIFY_PASSWORD [--blob ./path/to/blob]")
		return
	}

	if err != nil {
		fmt.Println("Error logging in: ", err)
		os.Exit(1)
		return
	}

	// Command loop
	reader := bufio.NewReader(os.Stdin)

	printHelp()

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		cmds := strings.Split(strings.TrimSpace(text), " ")

		switch cmds[0] {
		case "track":
			if len(cmds) < 2 {
				fmt.Println("You must specify the Base62 Spotify ID of the track")
			} else {
				funcTrack(session, cmds[1])
			}

		case "playlists":
			funcPlaylists(session)

		case "play":
			if len(cmds) < 2 {
				fmt.Println("You must specify the Base62 Spotify ID of the track")
			} else {
				funcPlay(session, cmds[1])
			}
		}
	}
}

func funcTrack(session *core.Session, trackId string) {
	fmt.Println("Loading track: ", trackId)

	track, err := session.Mercury().GetTrack(utils.Base62ToHex(trackId))
	if err != nil {
		fmt.Println("Error loading track: ", err)
		return
	}

	fmt.Println("Track title: ", track.GetName())
}

func funcPlaylists(session *core.Session) {
	fmt.Println("Listing playlists")

	playlist, err := session.Mercury().GetRootPlaylist(session.Username())

	if err != nil || playlist.Contents == nil {
		fmt.Println("Error getting root list: ", err)
		return
	}

	items := playlist.Contents.Items
	for i := 0; i < len(items); i++ {
		id := strings.TrimPrefix(items[i].GetUri(), "spotify:")
		id = strings.Replace(id, ":", "/", -1)
		list, _ := session.Mercury().GetPlaylist(id)
		fmt.Println(list.Attributes.GetName(), id)

		if list.Contents != nil {
			for j := 0; j < len(list.Contents.Items); j++ {
				item := list.Contents.Items[j]
				fmt.Println(" ==> ", *item.Uri)
			}
		}
	}
}

func funcPlay(session *core.Session, trackId string) {
	fmt.Println("Loading track for play: ", trackId)

	// Get the track metadata: it holds information about which files and encodings are available
	track, err := session.Mercury().GetTrack(utils.Base62ToHex(trackId))
	if err != nil {
		fmt.Println("Error loading track: ", err)
		return
	}

	fmt.Println("Track:", track.GetName())

	// As a demo, select the OGG 160kbps variant of the track. The "high quality" setting in the official Spotify
	// app is the OGG 320kbps variant.
	var selectedFileId []byte
	for _, file := range track.GetFile() {
		if file.GetFormat() == Spotify.AudioFile_OGG_VORBIS_160 {
			fmt.Println("Selected OGG 160, id:", file.GetFileId())
			selectedFileId = file.GetFileId()
		}
	}

	// Synchronously load the track
	audioFile, err := session.Player().LoadTrack(track.GetGid(), selectedFileId)

	if err != nil {
		fmt.Printf("Error while loading track: %s\n", err)
	} else {
		// We have the track audio, let's play it!
		fmt.Println("Writing audio file")
		ioutil.WriteFile("/tmp/audio.ogg", audioFile.Data, 0644)
	}
}

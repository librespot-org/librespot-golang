package main

import (
	"Spotify"
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"librespot"
	"librespot/core"
	"librespot/utils"
	"os"
	"strings"
)

const defaultdevicename = "librespot"

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("load <track1> [...more tracks]: load tracks by spotify base 62 id")
	fmt.Println("hello:                          ask devices to identify themselves")
	fmt.Println("play:                           play current track")
	fmt.Println("pause:                          pause playing track")
	fmt.Println("devices:                        list availbale devices")
	fmt.Println("mdns:                           show devices found via zeroconf, and login on device")
	fmt.Println("playlist <playlist id>:         load tracks from given playlist")
	fmt.Println("rootlist:                       show list of user's playlists")
	fmt.Println("help:                           show this list\n")
}

func main() {
	// Read flags from commandline
	username := flag.String("username", "", "spotify username")
	password := flag.String("password", "", "spotify password")
	blob := flag.String("blob", "", "spotify auth blob")
	devicename := flag.String("devicename", defaultdevicename, "name of device")
	flag.Parse()

	// Authenticate
	var session *core.Session
	var err error

	if *username != "" && *password != "" {
		session, err = librespot.Login(*username, *password, *devicename)

		fmt.Println("Login blob: ", base64.StdEncoding.EncodeToString(session.ReusableAuthBlob()))
	} else if *blob != "" && *username != "" {
		blobBytes, err := base64.StdEncoding.DecodeString(*blob)

		if err != nil {
			fmt.Println("Invalid blob base64")
			return
		}

		session, err = librespot.LoginSaved(*username, blobBytes, *devicename)
	} else if os.Getenv("client_secret") != "" {
		session, err = librespot.LoginOAuth(*devicename, os.Getenv("client_id"), os.Getenv("client_secret"))
	} else {
		fmt.Println("need to supply a username and password or a blob file path")
		fmt.Println("./microclient --username SPOTIFY_USERNAME --blob ./path/to/blob")
		fmt.Println("or")
		fmt.Println("./microclient --username SPOTIFY_USERNAME --password SPOTIFY_PASSWORD")
		return
	}

	if err != nil {
		fmt.Println("Error logging in: ", err)
		return
	}

	// Command loop
	reader := bufio.NewReader(os.Stdin)
	/*sController := spirc.CreateController(session, session.ReusableAuthBlob())
	sController.SendHello()*/

	// ident := *identFlag
	printHelp()

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		cmds := strings.Split(strings.TrimSpace(text), " ")

		switch {
		case cmds[0] == "track":
			fmt.Println("Loading track: ", cmds[1])

			track, err := session.Mercury().GetTrack(utils.Base62ToHex(cmds[1]))
			if err != nil {
				fmt.Println("Error loading track: ", err)
				continue
			}

			fmt.Println("Track title: ", track.GetName())

		case cmds[0] == "playlists":
			fmt.Println("Listing playlists")

			playlist, err := session.Mercury().GetRootPlaylist(session.Username())

			if err != nil || playlist.Contents == nil {
				fmt.Println("Error getting root list")
				break
			}
			items := playlist.Contents.Items
			for i := 0; i < len(items); i++ {
				id := strings.TrimPrefix(items[i].GetUri(), "spotify:")
				id = strings.Replace(id, ":", "/", -1)
				list, _ := session.Mercury().GetPlaylist(id)
				fmt.Println(list.Attributes.GetName(), id)

				for j := 0; j < len(list.Contents.Items); j++ {
					item := list.Contents.Items[j]
					fmt.Println(" ==> ", *item.Uri)
				}
			}

		case cmds[0] == "play":
			fmt.Println("Loading track for play: ", cmds[1])

			track, err := session.Mercury().GetTrack(utils.Base62ToHex(cmds[1]))
			if err != nil {
				fmt.Println("Error loading track: ", err)
				continue
			}

			fmt.Println("Track:", track.GetName())

			var selectedFileId []byte
			for _, file := range track.GetFile() {
				if file.GetFormat() == Spotify.AudioFile_OGG_VORBIS_160 {
					fmt.Println("Selected OGG 160, id:", file.GetFileId())
					selectedFileId = file.GetFileId()
				}
			}

			audioFile, err := session.Player().LoadTrack(track.GetGid(), selectedFileId)

			if err != nil {
				fmt.Printf("Error while loading track: %s\n", err)
			} else {
				fmt.Println("Writing audio file")
				fmt.Printf("%x\n", audioFile.Data[0:512])
				ioutil.WriteFile("/tmp/audio.ogg", audioFile.Data, 0644)
			}
		}
	}
}

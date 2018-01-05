package main

import (
	"bufio"
	"flag"
	"fmt"
	"librespot"
	"librespot/core"
	"librespot/spirc"
	"os"
	"strconv"
	"strings"
)

const defaultdevicename = "librespot"

func chooseDevice(controller *spirc.Controller, reader *bufio.Reader) string {
	devices := controller.ListDevices()
	if len(devices) == 0 {
		fmt.Println("no devices")
		return ""
	}

	fmt.Println("\n choose a device:")
	for i, d := range devices {
		fmt.Printf("%v) %v %v \n", i, d.Name, d.Ident)
	}

	for {
		fmt.Print("Enter device number: ")
		text, _ := reader.ReadString('\n')
		i, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && i < len(devices) && i >= 0 {
			return devices[i].Ident
		}
		fmt.Println("invalid device number")

	}
}

func getDevice(controller *spirc.Controller, ident string, reader *bufio.Reader) string {
	if ident != "" {
		return ident
	} else {
		return chooseDevice(controller, reader)
	}
}

func addMdns(controller *spirc.Controller, reader *bufio.Reader) {
	devices, err := controller.ListMdnsDevices()
	if err != nil {
		fmt.Println("Mdns devices can only be found when micro-controller is started \n" +
			"in discovery mode.  Restart without a username and password and with a --blobPath \n" +
			"argument (path where discovery blob will be saved) to start micro-controller in \n" +
			"disocvery mode \n")
		return
	}

	if len(devices) == 0 {
		fmt.Println("no devices found")
		return
	}
	fmt.Println("\n choose a device:")
	for i, d := range devices {
		fmt.Printf("%v) [mdns]%v %v \n", i, d.Name, d.Url)
	}
	var url string
	for {
		fmt.Print("Enter device number: ")
		text, _ := reader.ReadString('\n')
		i, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && i < len(devices) && i >= 0 {
			url = devices[i].Url
			break
		}
		fmt.Println("invalid device number")
	}

	controller.ConnectToDevice(url)

}

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
	username := flag.String("username", "", "spotify username")
	password := flag.String("password", "", "spotify password")
	blobPath := flag.String("blobPath", "", "path to saved blob")
	identFlag := flag.String("ident", "", "intially selected ident")
	devicename := flag.String("devicename", defaultdevicename, "name of device")
	flag.Parse()

	var session *core.Session
	var err error

	if *username != "" && *password != "" {
		session, err = librespot.Login(*username, *password, *devicename)
	} else if *blobPath != "" {
		if _, err = os.Stat(*blobPath); os.IsNotExist(err) {
			session, err = librespot.LoginDiscovery(*blobPath, *devicename)
		} else {
			session, err = librespot.LoginDiscoveryBlobFile(*blobPath, *devicename)
		}
	} else if os.Getenv("client_secret") != "" {
		session, err = librespot.LoginOAuth(*devicename, os.Getenv("client_id"), os.Getenv("client_secret"))
	} else {
		fmt.Println("need to supply a username and password or a blob file path")
		fmt.Println("./spirccontroller --blobPath ./path/to/blob")
		fmt.Println("or")
		fmt.Println("./spirccontroller --username SPOTIFY_USERNAME --password SPOTIFY_PASSWORD")
		return
	}

	if err != nil {
		fmt.Println("Error logging in: ", err)
		return
	}

	sController := spirc.CreateController(session, session.ReusableAuthBlob())

	reader := bufio.NewReader(os.Stdin)
	ident := *identFlag
	printHelp()
	for {
		fmt.Print("Enter a command: ")
		text, _ := reader.ReadString('\n')
		cmds := strings.Split(strings.TrimSpace(text), " ")

		switch {
		case cmds[0] == "load":
			ident = getDevice(sController, ident, reader)
			if ident != "" {
				sController.LoadTrack(ident, cmds[1:])
			}
		case cmds[0] == "hello":
			sController.SendHello()
		case cmds[0] == "play":
			ident = getDevice(sController, ident, reader)
			if ident != "" {
				sController.SendPlay(ident)
			}
		case cmds[0] == "pause":
			ident = getDevice(sController, ident, reader)
			if ident != "" {
				sController.SendPause(ident)
			}
		case cmds[0] == "devices":
			ident = chooseDevice(sController, reader)
		case cmds[0] == "mdns":
			addMdns(sController, reader)
		case cmds[0] == "help":
			printHelp()
		case cmds[0] == "playlist":
			playlist, err := session.Mercury().GetPlaylist(cmds[1])
			if err != nil || playlist.Contents == nil {
				fmt.Println("Playlist not found")
				break
			}
			items := playlist.Contents.Items
			var ids []string
			for i := 0; i < len(items); i++ {
				id := strings.TrimPrefix(items[i].GetUri(), "spotify:track:")
				ids = append(ids, id)
			}
			ident = getDevice(sController, ident, reader)
			if ident != "" {
				sController.LoadTrack(ident, ids)
			}
		case cmds[0] == "rootlist":
			playlist, _ := session.Mercury().GetRootPlaylist(session.Username())
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
			}
		}
	}

}

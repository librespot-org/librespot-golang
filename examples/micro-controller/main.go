package main

import (
	"bufio"
	"fmt"
	"github.com/badfortrains/spotcontrol"
	"strings"
	"strconv"
	"flag"
	"os"
)

func chooseDevice(controller *spotcontrol.SpircController, reader *bufio.Reader) string{
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
		if err == nil && i < len(devices) && i >= 0{
			return devices[i].Ident
		}
		fmt.Println("invalid device number")

	}
}

func getDevice(controller *spotcontrol.SpircController, ident string, reader *bufio.Reader) string{
	if ident != "" {
		return ident
	} else {
		return chooseDevice(controller, reader)
	}
}

func addMdns(controller *spotcontrol.SpircController, reader *bufio.Reader) {
	devices := controller.ListMdnsDevices()
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
		if err == nil && i < len(devices) && i >= 0{
			url = devices[i].Url
			break
		}
		fmt.Println("invalid device number")
	}

	controller.ConnectToDevice(url)

}

func printHelp(){
	fmt.Println("\nAvailable commands:")
	fmt.Println("load <track1> [...more tracks]: load tracks by spotify base 62 id")
	fmt.Println("hello:                          ask devices to identify themselves")
	fmt.Println("play:                           play current track")
	fmt.Println("pause:                          pause playing track")
	fmt.Println("devices:                        list availbale devices")
	fmt.Println("mdns:                           show devices found via zeroconf, and login on device\n")
	fmt.Println("help:                           show this list\n")
}


func main() {
	username := flag.String("username", "", "spotify username")
	password := flag.String("password", "", "spotify password")
	appkey := flag.String("appkey", "./spotify_appkey.key", "spotify appkey file path")
	blobPath := flag.String("blobPath", "", "path to saved blob")
	flag.Parse()

	var sController *spotcontrol.SpircController
	if *username != "" && *password != ""{
		sController = spotcontrol.Login(*username, *password, *appkey)
	} else if *blobPath != "" {
		if _, err := os.Stat(*blobPath); os.IsNotExist(err) {
			sController = spotcontrol.LoginDiscovery(*blobPath, *appkey)
		} else {
			sController = spotcontrol.LoginBlobFile(*blobPath, *appkey)
		}
	} else {
		fmt.Println("need to supply a username and password or a blob file path")
		fmt.Println("./spirccontroller --blobPath ./path/to/blob")
		fmt.Println("or")
		fmt.Println("./spirccontroller --username SPOTIFY_USERNAME --password SPOTIFY_PASSWORD")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	var ident string
	printHelp()
	for {
		fmt.Print("Enter a command: ")
		text, _ := reader.ReadString('\n')
		cmds := strings.Split(strings.TrimSpace(text),  " ")

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
		}
	}

}
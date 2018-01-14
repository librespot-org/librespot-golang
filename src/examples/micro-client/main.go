package main

import (
	"Spotify"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/xlab/portaudio-go/portaudio"
	"github.com/xlab/vorbis-go/decoder"
	"io/ioutil"
	"librespot"
	"librespot/core"
	"librespot/utils"
	"log"
	"os"
	"strings"
	"sync"
	"unsafe"
)

const (
	// The device name that is registered to Spotify servers
	kDefaultDeviceName = "librespot"
	// The number of samples per channel in the decoded audio
	kSamplesPerChannel = 2048
	// The samples bit depth
	kBitDepth = 16
	// The samples format
	kSampleFormat = portaudio.PaFloat32
)

func main() {
	// First, initialize PortAudio
	if err := portaudio.Initialize(); paError(err) {
		log.Fatalln("PortAudio init error: ", paErrorText(err))
	}

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

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("play <track>:                   play specified track by spotify base64 id")
	fmt.Println("track <track>:                  show details on specified track by spotify base64 id")
	fmt.Println("playlists:                      show your playlist")
	fmt.Println("help:                           show this list")
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
		// We have the track audio, let's play it! Initialize the OGG decoder, and start a PortAudio stream.
		// Note that we skip the first 167 bytes as it is a Spotify-specific header. You can decode it by
		// using this: https://sourceforge.net/p/despotify/code/HEAD/tree/java/trunk/src/main/java/se/despotify/client/player/SpotifyOggHeader.java
		fmt.Println("Setting up OGG decoder...")
		dec, err := decoder.New(bytes.NewReader(audioFile.Data[167:]), kSamplesPerChannel)
		if err != nil {
			log.Fatalln(err)
		}

		info := dec.Info()

		go func() {
			dec.Decode()
			dec.Close()
		}()

		fmt.Println("Setting up PortAudio stream...")
		fmt.Printf("PortAudio channels: %d / SampleRate: %f / Samples out: %d\n", info.Channels, info.SampleRate, dec.SamplesOut())
		channels := int32(2)

		var wg sync.WaitGroup
		var stream *portaudio.Stream
		callback := paCallback(&wg, int(channels), dec.SamplesOut())

		if err := portaudio.OpenDefaultStream(&stream, 0, channels, kSampleFormat, 44100,
			kSamplesPerChannel, callback, nil); paError(err) {
			log.Fatalln(paErrorText(err))
		}

		fmt.Println("Starting playback...")
		if err := portaudio.StartStream(stream); paError(err) {
			log.Fatalln(paErrorText(err))
		}

		wg.Wait()
	}
}

// PortAudio helpers
func paError(err portaudio.Error) bool {
	return portaudio.ErrorCode(err) != portaudio.PaNoError

}

func paErrorText(err portaudio.Error) string {
	return "PortAudio error: " + portaudio.GetErrorText(err)
}

func paCallback(wg *sync.WaitGroup, channels int, samples <-chan [][]float32) portaudio.StreamCallback {
	wg.Add(1)
	return func(_ unsafe.Pointer, output unsafe.Pointer, sampleCount uint,
		_ *portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags, _ unsafe.Pointer) int32 {

		const (
			statusContinue = int32(portaudio.PaContinue)
			statusComplete = int32(portaudio.PaComplete)
		)

		frame, ok := <-samples
		if !ok {
			wg.Done()
			return statusComplete
		}
		if len(frame) > int(sampleCount) {
			frame = frame[:sampleCount]
		}

		var idx int
		out := (*(*[1 << 32]float32)(unsafe.Pointer(output)))[:int(sampleCount)*channels]
		for _, sample := range frame {
			if len(sample) > channels {
				sample = sample[:channels]
			}
			for i := range sample {
				out[idx] = sample[i]
				idx++
			}
		}

		return statusContinue
	}
}

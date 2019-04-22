package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"unsafe"

	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot"
	"github.com/librespot-org/librespot-golang/librespot/core"
	"github.com/librespot-org/librespot-golang/librespot/utils"
	"github.com/xlab/portaudio-go/portaudio"
	"github.com/xlab/vorbis-go/decoder"
)

const (
	// The device name that is registered to Spotify servers
	defaultDeviceName = "librespot"
	// The number of samples per channel in the decoded audio
	samplesPerChannel = 2048
	// The samples bit depth
	bitDepth = 16
	// The samples format
	sampleFormat = portaudio.PaFloat32
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
	devicename := flag.String("devicename", defaultDeviceName, "name of device")
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
		case "help":
			printHelp()

		case "track":
			if len(cmds) < 2 {
				fmt.Println("You must specify the Base62 Spotify ID of the track")
			} else {
				funcTrack(session, cmds[1])
			}

		case "artist":
			if len(cmds) < 2 {
				fmt.Println("You must specify the Base62 Spotify ID of the artist")
			} else {
				funcArtist(session, cmds[1])
			}

		case "album":
			if len(cmds) < 2 {
				fmt.Println("You must specify the Base62 Spotify ID of the album")
			} else {
				funcAlbum(session, cmds[1])
			}

		case "playlists":
			funcPlaylists(session)

		case "search":
			funcSearch(session, cmds[1])

		case "play":
			if len(cmds) < 2 {
				fmt.Println("You must specify the Base62 Spotify ID of the track")
			} else {
				funcPlay(session, cmds[1])
			}

		default:
			fmt.Println("Unknown command")
		}
	}
}

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("play <track>:                   play specified track by spotify base62 id")
	fmt.Println("track <track>:                  show details on specified track by spotify base62 id")
	fmt.Println("album <album>:                  show details on specified album by spotify base62 id")
	fmt.Println("artist <artist>:                show details on specified artist by spotify base62 id")
	fmt.Println("search <keyword>:               start a search on the specified keyword")
	fmt.Println("playlists:                      show your playlists")
	fmt.Println("help:                           show this help")
}

func funcTrack(session *core.Session, trackID string) {
	fmt.Println("Loading track: ", trackID)

	track, err := session.Mercury().GetTrack(utils.Base62ToHex(trackID))
	if err != nil {
		fmt.Println("Error loading track: ", err)
		return
	}

	fmt.Println("Track title: ", track.GetName())
}

func funcArtist(session *core.Session, artistID string) {
	artist, err := session.Mercury().GetArtist(utils.Base62ToHex(artistID))
	if err != nil {
		fmt.Println("Error loading artist:", err)
		return
	}

	fmt.Printf("Artist: %s\n", artist.GetName())
	fmt.Printf("Popularity: %d\n", artist.GetPopularity())
	fmt.Printf("Genre: %s\n", artist.GetGenre())

	if artist.GetTopTrack() != nil && len(artist.GetTopTrack()) > 0 {
		// Spotify returns top tracks in multiple countries. We take the first
		// one as example, but we should use the country data returned by the
		// Spotify server (session.Country())
		tt := artist.GetTopTrack()[0]
		fmt.Printf("\nTop tracks (country %s):\n", tt.GetCountry())

		for _, t := range tt.GetTrack() {
			// To save bandwidth, only track IDs are returned. If you want
			// the track name, you need to fetch it.
			fmt.Printf(" => %s\n", utils.ConvertTo62(t.GetGid()))
		}
	}

	fmt.Printf("\nAlbums:\n")
	for _, ag := range artist.GetAlbumGroup() {
		for _, a := range ag.GetAlbum() {
			fmt.Printf(" => %s\n", utils.ConvertTo62(a.GetGid()))
		}
	}

}

func funcAlbum(session *core.Session, albumID string) {
	album, err := session.Mercury().GetAlbum(utils.Base62ToHex(albumID))
	if err != nil {
		fmt.Println("Error loading album:", err)
		return
	}

	fmt.Printf("Album: %s\n", album.GetName())
	fmt.Printf("Popularity: %d\n", album.GetPopularity())
	fmt.Printf("Genre: %s\n", album.GetGenre())
	fmt.Printf("Date: %d-%d-%d\n", album.GetDate().GetYear(), album.GetDate().GetMonth(), album.GetDate().GetDay())
	fmt.Printf("Label: %s\n", album.GetLabel())
	fmt.Printf("Type: %s\n", album.GetTyp())

	fmt.Printf("Artists: ")
	for _, artist := range album.GetArtist() {
		fmt.Printf("%s ", utils.ConvertTo62(artist.GetGid()))
	}
	fmt.Printf("\n")

	for _, disc := range album.GetDisc() {
		fmt.Printf("\nDisc %d (%s): \n", disc.GetNumber(), disc.GetName())

		for _, track := range disc.GetTrack() {
			fmt.Printf(" => %s\n", utils.ConvertTo62(track.GetGid()))
		}
	}

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

func funcSearch(session *core.Session, keyword string) {
	resp, err := session.Mercury().Search(keyword, 12, session.Country(), session.Username())

	if err != nil {
		fmt.Println("Failed to search:", err)
		return
	}

	res := resp.Results

	fmt.Println("Search results for ", keyword)
	fmt.Println("=============================")

	if res.Error != nil {
		fmt.Println("Search result error:", res.Error)
	}

	fmt.Printf("Albums: %d (total %d)\n", len(res.Albums.Hits), res.Albums.Total)

	for _, album := range res.Albums.Hits {
		fmt.Printf(" => %s (%s)\n", album.Name, album.Uri)
	}

	fmt.Printf("\nArtists: %d (total %d)\n", len(res.Artists.Hits), res.Artists.Total)

	for _, artist := range res.Artists.Hits {
		fmt.Printf(" => %s (%s)\n", artist.Name, artist.Uri)
	}

	fmt.Printf("\nTracks: %d (total %d)\n", len(res.Tracks.Hits), res.Tracks.Total)

	for _, track := range res.Tracks.Hits {
		fmt.Printf(" => %s (%s)\n", track.Name, track.Uri)
	}
}

func funcPlay(session *core.Session, trackID string) {
	fmt.Println("Loading track for play: ", trackID)

	// Get the track metadata: it holds information about which files and encodings are available
	track, err := session.Mercury().GetTrack(utils.Base62ToHex(trackID))
	if err != nil {
		fmt.Println("Error loading track: ", err)
		return
	}

	fmt.Println("Track:", track.GetName())

	// As a demo, select the OGG 160kbps variant of the track. The "high quality" setting in the official Spotify
	// app is the OGG 320kbps variant.
	var selectedFile *Spotify.AudioFile
	for _, file := range track.GetFile() {
		if file.GetFormat() == Spotify.AudioFile_OGG_VORBIS_160 {
			selectedFile = file
		}
	}

	// Synchronously load the track
	audioFile, err := session.Player().LoadTrack(selectedFile, track.GetGid())

	// TODO: channel to be notified of chunks downloaded (or reader?)

	if err != nil {
		fmt.Printf("Error while loading track: %s\n", err)
	} else {
		// We have the track audio, let's play it! Initialize the OGG decoder, and start a PortAudio stream.
		// Note that we skip the first 167 bytes as it is a Spotify-specific header. You can decode it by
		// using this: https://sourceforge.net/p/despotify/code/HEAD/tree/java/trunk/src/main/java/se/despotify/client/player/SpotifyOggHeader.java
		fmt.Println("Setting up OGG decoder...")
		dec, err := decoder.New(audioFile, samplesPerChannel)
		if err != nil {
			log.Fatalln(err)
		}

		info := dec.Info()

		go func() {
			dec.Decode()
			dec.Close()
		}()

		fmt.Println("Setting up PortAudio stream...")
		fmt.Printf("PortAudio channels: %d / SampleRate: %f\n", info.Channels, info.SampleRate)

		var wg sync.WaitGroup
		var stream *portaudio.Stream
		callback := paCallback(&wg, int(info.Channels), dec.SamplesOut())

		if err := portaudio.OpenDefaultStream(&stream, 0, info.Channels, sampleFormat, info.SampleRate,
			samplesPerChannel, callback, nil); paError(err) {
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

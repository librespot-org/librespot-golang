package discovery

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/badfortrains/mdns"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/librespot-org/librespot-golang/librespot/crypto"
	"github.com/librespot-org/librespot-golang/librespot/utils"
	"net"
)

// connectInfo stores the information about Spotify Connect connection
type connectInfo struct {
	DeviceID  string `json:"deviceID"`
	PublicKey string `json:"publicKey"`
}

// connectDeviceMdns stores the information about Spotify Connect MDNS Request
type connectDeviceMdns struct {
	Path string
	Name string
}

// connectGetInfo stores the information about a Spotify Connect information Request
type connectGetInfo struct {
	Status           int    `json:"status"`
	StatusError      string `json:"statusError"`
	SpotifyError     int    `json:"spotifyError"`
	Version          string `json:"version"`
	DeviceID         string `json:"deviceID"`
	RemoteName       string `json:"remoteName"`
	ActiveUser       string `json:"activeUser"`
	PublicKey        string `json:"publicKey"`
	DeviceType       string `json:"deviceType"`
	LibraryVersion   string `json:"libraryVersion"`
	AccountReq       string `json:"accountReq"`
	BrandDisplayName string `json:"brandDisplayName"`
	ModelDisplayName string `json:"modelDisplayName"`
}

// Discovery stores the information about Spotify Connect Discovery Request
type Discovery struct {
	keys       crypto.PrivateKeys
	cachePath  string
	loginBlob  utils.BlobInfo
	deviceId   string
	deviceName string

	mdnsServer  *mdns.Server
	httpServer  *http.Server
	devices     []connectDeviceMdns
	devicesLock sync.RWMutex
}

// makeConnectGetInfo builds a connectGetInfo structure with the provided values
func makeConnectGetInfo(deviceId string, deviceName string, publicKey string) connectGetInfo {
	return connectGetInfo{
		Status:           101,
		StatusError:      "ERROR-OK",
		SpotifyError:     0,
		Version:          "1.3.0",
		DeviceID:         deviceId,
		RemoteName:       deviceName,
		ActiveUser:       "",
		PublicKey:        publicKey,
		DeviceType:       "UNKNOWN",
		LibraryVersion:   "0.1.0",
		AccountReq:       "PREMIUM",
		BrandDisplayName: "librespot",
		ModelDisplayName: "librespot",
	}
}

func blobFromDiscovery(deviceName string) *utils.BlobInfo {
	deviceId := utils.GenerateDeviceId(deviceName)
	d := LoginFromConnect("", deviceId, deviceName)
	return &d.loginBlob
}

// Advertises a Spotify service via mdns. It waits for the user to connect to 'librespot' device, extracts login data
// and returns the resulting login BlobInfo.
func LoginFromConnect(cachePath string, deviceId string, deviceName string) *Discovery {
	d := Discovery{
		keys:       crypto.GenerateKeys(),
		cachePath:  cachePath,
		deviceId:   deviceId,
		deviceName: deviceName,
	}

	done := make(chan int)

	l, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	go d.startHttp(done, l)
	d.startDiscoverable()

	<-done

	return &d
}

func CreateFromBlob(blob utils.BlobInfo, cachePath, deviceId string, deviceName string) *Discovery {
	d := Discovery{
		keys:       crypto.GenerateKeys(),
		cachePath:  cachePath,
		deviceId:   deviceId,
		loginBlob:  blob,
		deviceName: deviceName,
	}

	d.FindDevices()

	return &d
}

func CreateFromFile(cachePath, deviceId string, deviceName string) *Discovery {
	blob, err := utils.BlobFromFile(cachePath)
	if err != nil {
		log.Fatal("failed to get blob from file")
	}

	return CreateFromBlob(blob, cachePath, deviceId, deviceName)
}

func (d *Discovery) DeviceId() string {
	return d.deviceId
}

func (d *Discovery) DeviceName() string {
	return d.deviceName
}

func (d *Discovery) LoginBlob() utils.BlobInfo {
	return d.loginBlob
}

// Devices return an immutable copy of the current MDNS-discovered devices, thread-safe
func (d *Discovery) Devices() []connectDeviceMdns {
	res := make([]connectDeviceMdns, 0, len(d.devices))
	return append(res, d.devices...)
}

func (d *Discovery) FindDevices() {
	ch := make(chan *mdns.ServiceEntry, 10)

	d.devices = make([]connectDeviceMdns, 0)
	go func() {
		for entry := range ch {
			cPath := findCpath(entry.InfoFields)
			path := fmt.Sprintf("http://%v:%v%v", entry.AddrV4, entry.Port, cPath)
			fmt.Println("Found a device", entry)
			d.devicesLock.Lock()
			d.devices = append(d.devices, connectDeviceMdns{
				Path: path,
				Name: strings.Replace(entry.Name, "._spotify-connect._tcp.local.", "", 1),
			})
			fmt.Println("devices", d.devices)
			d.devicesLock.Unlock()
		}
		fmt.Println("closed")
	}()

	err := mdns.Lookup("_spotify-connect._tcp.", ch)
	if err != nil {
		log.Fatal("lookup error", err)
	}
}

func (d *Discovery) ConnectToDevice(address string) {
	resp, err := http.Get(address + "?action=connectGetInfo")
	resp, err = http.Get(address + "?action=resetUsers")
	resp, err = http.Get(address + "?action=connectGetInfo")

	fmt.Println("start get")
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	info := connectInfo{}
	err = decoder.Decode(&info)
	if err != nil {
		panic("bad json")
	}
	fmt.Println("resposne", resp)

	client64 := base64.StdEncoding.EncodeToString(d.keys.PubKey())
	blob, err := d.loginBlob.MakeAuthBlob(info.DeviceID,
		info.PublicKey, d.keys)
	if err != nil {
		panic("bad blob")
	}

	body := makeAddUserRequest(d.loginBlob.Username, blob, client64, d.deviceId, d.deviceName)
	resp, err = http.PostForm(address, body)
	defer resp.Body.Close()
	decoder = json.NewDecoder(resp.Body)
	var f interface{}
	err = decoder.Decode(&f)

	fmt.Println("got", f, resp, err)
}

func makeAddUserRequest(username string, blob string, key string, deviceId string, deviceName string) url.Values {
	v := url.Values{}
	v.Set("action", "addUser")
	v.Add("userName", username)
	v.Add("blob", blob)
	v.Add("clientKey", key)
	v.Add("deviceId", deviceId)
	v.Add("deviceName", deviceName)
	return v
}

func findCpath(info []string) string {
	for _, i := range info {
		if strings.Contains(i, "CPath") {
			return strings.Split(i, "=")[1]
		}
	}
	return ""
}

func (d *Discovery) handleAddUser(r *http.Request) error {
	//already have login info, ignore
	if d.loginBlob.Username != "" {
		return nil
	}

	username := r.FormValue("userName")
	client64 := r.FormValue("clientKey")
	blob64 := r.FormValue("blob")

	if username == "" || client64 == "" || blob64 == "" {
		log.Println("Bad Request, addUser")
		return errors.New("bad username Request")
	}

	blob, err := utils.NewBlobInfo(blob64, client64, d.keys,
		d.deviceId, username)
	if err != nil {
		return errors.New("failed to decode blob")
	}

	err = blob.SaveToFile(d.cachePath)
	if err != nil {
		log.Println("failed to cache login info")
	}

	d.loginBlob = blob
	d.mdnsServer.Shutdown()
	return nil
}

func (d *Discovery) startHttp(done chan int, l net.Listener) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		action := r.FormValue("action")
		fmt.Println("got Request: ", action)
		switch {
		case "connectGetInfo" == action || "resetUsers" == action:
			client64 := base64.StdEncoding.EncodeToString(d.keys.PubKey())
			info := makeConnectGetInfo(d.deviceId, d.deviceName, client64)

			js, err := json.Marshal(info)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		case "addUser" == action:
			err := d.handleAddUser(r)
			if err == nil {
				done <- 1
			}
		}
	})

	d.httpServer = &http.Server{}
	err := d.httpServer.Serve(l)
	if err != nil {
		fmt.Println("got an error", err)
	}
}

func (d *Discovery) startDiscoverable() {
	fmt.Println("start discoverable")
	info := []string{"VERSION=1.0", "CPath=/"}

	ifaces, err := net.Interfaces()
	// Handle err
	ips := make([]net.IP, 0)
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// Handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ips = append(ips, v.IP)
			case *net.IPAddr:
				ips = append(ips, v.IP)
			}
			fmt.Println("found ip ", ips)
			// process IP address
		}
	}

	service, err := mdns.NewMDNSService("librespot"+strconv.Itoa(rand.Intn(200)),
		"_spotify-connect._tcp", "", "", 8000, ips, info)
	if err != nil {
		fmt.Println(err)
		log.Fatal("error starting Discovery")
	}
	server, err := mdns.NewServer(&mdns.Config{
		Zone: service,
	})
	if err != nil {
		log.Fatal("error starting Discovery")
	}
	d.mdnsServer = server
}

package spotcontrol

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

	"net"
)

type connectInfo struct {
	DeviceID  string `json:"deviceID"`
	PublicKey string `json:"publicKey"`
}

type connectDeviceMdns struct {
	path string
	name string
}

type getInfo struct {
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

type discovery struct {
	keys       privateKeys
	cachePath  string
	loginBlob  BlobInfo
	deviceId   string
	deviceName string

	mdnsServer  *mdns.Server
	httpServer  *http.Server
	devices     []connectDeviceMdns
	devicesLock sync.RWMutex
}

func makeGetInfo(deviceId, deviceName, publicKey string) getInfo {
	return getInfo{
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

// Advertise spotify service via mdns.  Waits for user
// to connect to 'spotcontrol' device.  Extracts login data
// and returns login BlobInfo.
func BlobFromDiscovery(deviceName string) *BlobInfo {
	deviceId := generateDeviceId(deviceName)
	d := loginFromConnect("", deviceId, deviceName)
	return &d.loginBlob
}

func loginFromConnect(cachePath, deviceId string, deviceName string) *discovery {
	d := discovery{
		keys:       generateKeys(),
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

func discoveryFromBlob(blob BlobInfo, cachePath, deviceId string, deviceName string) *discovery {
	d := discovery{
		keys:       generateKeys(),
		cachePath:  cachePath,
		deviceId:   deviceId,
		loginBlob:  blob,
		deviceName: deviceName,
	}

	d.FindDevices()

	return &d
}

func loginFromFile(cachePath, deviceId string, deviceName string) *discovery {
	blob, err := blobFromFile(cachePath)
	if err != nil {
		log.Fatal("failed to get blob from file")
	}

	return discoveryFromBlob(blob, cachePath, deviceId, deviceName)
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

func (d *discovery) FindDevices() {
	ch := make(chan *mdns.ServiceEntry, 10)

	d.devices = make([]connectDeviceMdns, 0)
	go func() {
		for entry := range ch {
			cPath := findCpath(entry.InfoFields)
			url := fmt.Sprintf("http://%v:%v%v", entry.AddrV4, entry.Port, cPath)
			fmt.Println("Found a device", entry)
			d.devicesLock.Lock()
			d.devices = append(d.devices, connectDeviceMdns{
				path: url,
				name: strings.Replace(entry.Name, "._spotify-connect._tcp.local.", "", 1),
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

func (d *discovery) ConnectToDevice(address string) {
	resp, err := http.Get(address + "?action=getInfo")
	resp, err = http.Get(address + "?action=resetUsers")
	resp, err = http.Get(address + "?action=getInfo")

	fmt.Println("start get")
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	info := connectInfo{}
	err = decoder.Decode(&info)
	if err != nil {
		panic("bad json")
	}
	fmt.Println("resposne", resp)

	client64 := base64.StdEncoding.EncodeToString(d.keys.pubKey())
	blob, err := d.loginBlob.makeAuthBlob(info.DeviceID,
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

func (d *discovery) handleAddUser(r *http.Request) error {
	//already have login info, ignore
	if d.loginBlob.Username != "" {
		return nil
	}

	username := r.FormValue("userName")
	client64 := r.FormValue("clientKey")
	blob64 := r.FormValue("blob")

	if username == "" || client64 == "" || blob64 == "" {
		log.Println("Bad request, addUser")
		return errors.New("Bad username request")
	}

	blob, err := newBlobInfo(blob64, client64, d.keys,
		d.deviceId, username)
	if err != nil {
		return errors.New("failed to decode blob")
	}

	err = blob.saveToFile(d.cachePath)
	if err != nil {
		log.Println("failed to cache login info")
	}

	d.loginBlob = blob
	d.mdnsServer.Shutdown()
	return nil
}

func (d *discovery) startHttp(done chan int, l net.Listener) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		action := r.FormValue("action")
		fmt.Println("got request: ", action)
		switch {
		case "getInfo" == action || "resetUsers" == action:
			client64 := base64.StdEncoding.EncodeToString(d.keys.pubKey())
			info := makeGetInfo(d.deviceId, d.deviceName, client64)

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

func (d *discovery) startDiscoverable() {
	fmt.Println("start discoverable")
	info := []string{"VERSION=1.0", "CPath=/"}

	ifaces, err := net.Interfaces()
	// handle err
	ips := make([]net.IP, 0)
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
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

	service, err := mdns.NewMDNSService("spotcontrol"+strconv.Itoa(rand.Intn(200)),
		"_spotify-connect._tcp", "", "", 8000, ips, info)
	if err != nil {
		fmt.Println(err)
		log.Fatal("error starting discovery")
	}
	server, err := mdns.NewServer(&mdns.Config{
		Zone: service,
	})
	if err != nil {
		log.Fatal("error starting discovery")
	}
	d.mdnsServer = server
}

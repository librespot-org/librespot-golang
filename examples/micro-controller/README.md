# micro-controller

Provides very basic (and incomplete) command line interface for controlling spotify connect devices.  
Currently supports loading tracks, playing and pausing.

##usage
````
//in micro-controller directory
./micro-controller --username SPOTIFY_USERNAME --password SPOTIFY_PASSWORD --appkey PATH_TO_APPKEY

//load tracks using space deliminated list of spotify track ids
load 3Vn9oCZbdI1EMO7jxdz2Rc 2nMW1mZmdIt5rZCsX1uh9J
````
Where PATH_TO_APPKEY is the path to your Spotify application key file (defaults to ./spotify_appkey.key).
Spotify track ids can be found using the [spotify api console](https://developer.spotify.com/web-api/console/get-search-item/)

#### Mdns discovery
Instead of supplying a username and password, the micro controller can advertise itself as a spotify connect device and extract the necessary authentication blob when connected to via the desktop or mobile spotify apps.  This authentication blob is then saved, and can be replayed to other spotify connect devices on the network.

````
//start in discovery mode
./micro-controller --blobPath PATH_TO_STORE_BLOB
````
Where PATH_TO_STORE_BLOB is the path of where to save the blob.  Once blob is saved it will be loaded on subsequent runs, and the connect step can be skipped.

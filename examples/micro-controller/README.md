# spotcontrol_example

Example program demonstrating the [spotcontrol library](https://github.com/badfortrains/spotcontrol).
Provides very basic (and incomplete) command line interface for controlling spotify connect devices.  
Currently supports loading tracks, playing and pausing.

## Install
`````
go get github.com/badfortrains/spotcontrol
go get github.com/badfortrains/spotcontrol_example

//in spotcontrol_example directory
go build
````
##usage
````
//in spotcontrol_example directory
./spotcontrol_example --username SPOTIFY_USERNAME --password SPOTIFY_PASSWORD --appkey PATH_TO_APPKEY

//load tracks using space deliminated list of spotify track ids
load 3Vn9oCZbdI1EMO7jxdz2Rc 2nMW1mZmdIt5rZCsX1uh9J
````
Where PATH_TO_APPKEY is the path to your Spotify application key file (defaults to ./spotify_appkey.key).
Spotify track ids can be found using the [spotify api console](https://developer.spotify.com/web-api/console/get-search-item/)



# micro-controller

Provides very basic (and incomplete) command line interface for controlling spotify connect devices.  
Currently supports loading tracks, playing and pausing.

##Install
In this directory, build with
````
go build
````

##Usage
````
//in micro-controller directory
./micro-controller --username SPOTIFY_USERNAME --password SPOTIFY_PASSWORD

//load tracks using space deliminated list of spotify track ids
load 3Vn9oCZbdI1EMO7jxdz2Rc 2nMW1mZmdIt5rZCsX1uh9J
````
Spotify track ids can be found using the [spotify api console](https://developer.spotify.com/web-api/console/get-search-item/)

#### Mdns discovery
Instead of supplying a username and password, the micro controller can advertise itself as a spotify connect device and extract the necessary authentication blob when connected to via the desktop or mobile spotify apps.  This authentication blob is then saved, and can be replayed to other spotify connect devices on the network.

````
//start in discovery mode
./micro-controller --blobPath PATH_TO_STORE_BLOB
````
Where PATH_TO_STORE_BLOB is the path where the blob will be saved.  Once blob is saved it will be loaded on subsequent runs, and the connect step can be skipped.

#### Oauth login 
Login using Oauth as described in the spotify [web api docs](https://developer.spotify.com/web-api/authorization-guide/#authorization_code_flow). User opens a url in their browser, signs in with Spotify and grants permission for the "streaming" scope. Spotify redirects the user to a local web server started by spotcontrol which extracts the authorization code and logs the user in. 
Requires: 

1. Register an app https://developer.spotify.com/my-applications
2. Copy your "client_id" and "client_secret" as enviroment variables

  ````
    export client_id=[Your id here]
    export client_secret=[Your client secret here]
  ````
3. Start spot control with no arguments



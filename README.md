## librespot-golang

### Introduction

librespot-golang is an opensource Golang library based on the [librespot](https://github.com/plietar/librespot) project,
allowing you to control Spotify Connect devices, get metadata, and play music. It has itself been based on
[SpotControl](https://github.com/badfortrains/spotcontrol), and its main goal is to provide a suitable replacement
for the defunct libspotify.

This is still highly experimental and in development. Do not use it in production projects yet, as the API is incomplete
and subject to heavy changes.

This fork contains changes that are more compatible with go.mod, while removing the Rust-esque package layout.
### Installation

This package can be installed using:

````
git clone github.com/librespot-org/librespot-golang
````

or by using it as a dependency in your godep vendor file.

### Usage

To use the package look at the example micro-controller (for Spotify Connect), or micro-client (for audio playback).

### Building for mobile

The package `librespotmobile` contains bindings suitable for use with Gomobile, which lets you use a subset of the
librespot library on Android and iOS.

To get started, install gomobile, and simply run (for Android):

````
cd /path/to/librespot-golang
export GOPATH=$(pwd)
gomobile init -ndk /path/to/android-ndk
gomobile bind librespotmobile
````

This will build you a file called `librespotmobile.aar` which you can include in your Android Studio project.

### To-Do's

* Handling disconnections, timeouts, etc (overall failure tolerance)
* Playlist management
* Spotify Radio support

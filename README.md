## librespot-golang

### Introduction

librespot-golang is an opensource Golang library based on the [librespot](https://github.com/plietar/librespot) project,
allowing you to control Spotify Connect devices, get metadata, and play music. It has itself been based on
[SpotControl](https://github.com/badfortrains/spotcontrol), and its main goal is to provide a suitable replacement
for the defunct libspotify.

This is still highly experimental and in development. Do not use it in production projects yet, as the API is incomplete
and subject to heavy changes.

### Installation

This package can be installed using:
````
go get github.com/librespot-org/librespot-golang
````

### Usage

To use the package look at the example micro-controller (for Spotify Connect), or micro-client (for audio playback).

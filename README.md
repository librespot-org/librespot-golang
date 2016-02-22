## spotcontrol

Spotcontrol is an open source library for controlling spotify connect devices based on the [librespot](https://github.com/plietar/librespot) project.  

Spotcontrol is a golang port of a small subset of the librespot functionality, focusing soley on controlling other devices, it does not offer any support for actual music playback.  A simple cli is included in the examples folder and demonstrates the key features of the library (loading tracks, playing, pausing).

### Instalation
This package can be installed using:
````
go get github.com/badfortrains/spotcontrol
````

### Usage
To use the package look at the example micro-controller, and see the godoc
````
go doc github.com/badfortrains/spotcontrol
````

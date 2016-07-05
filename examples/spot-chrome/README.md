## Spotcontrol chrome app

Basic chrome extension using [gopherjs](https://github.com/gopherjs/gopherjs) to translate spotcontrol to javascript.

Uses [Chrome tcp sockets](https://developer.chrome.com/apps/sockets_tcp) for network communication.

Currently supports "play", "pause", and setting device volume.

![Screenshot](https://raw.githubusercontent.com/badfortrains/spotcontrol/master/examples/spot-chrome/screenshot.png)

### Install

+ install [gopherjs](https://github.com/gopherjs/gopherjs) ````go get -u github.com/gopherjs/gopherjs````
+ compile to js. In spot-chrome directory: ````gopherjs build ````
+ Load the spot-chrome directory as an unpacked extension in chrome (https://developer.chrome.com/extensions/getstarted#unpacked)

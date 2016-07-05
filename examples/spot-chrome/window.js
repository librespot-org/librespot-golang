'use strict';

let appState = {
	devices: [],
	currentIdent: '',
	loggedIn: false
}

const PlayStatus = {
	0: 'Stopped',
	1: 'Playing',
	2: 'Paused',
	3: 'Loading'
}


function selectComponent(){
	function option(device){
		const state = device.device_state || {};
		return `<option ${appState.currentIdent == device.ident ? 'selected' : ''} 
								value=${device.ident}>${state.name || device.ident}</option>`;
	}

	return `
		<select>
			${appState.devices.map(option)}
		</select>
	`
}

function getDevice(ident) {
	return appState.devices.find(d => d.ident == ident);
}

function getCurrentDevice() {
	return appState.devices.find(d => d.ident == appState.currentIdent) 
}

function stateComponent(){
	const device = getCurrentDevice() || {}
	const state = device.state || {}
	const currentTrack = device.currentTrack || {}
  const deviceState = device.device_state || {}
	return `
		<ul>
			<li>is_active: ${deviceState.is_active} </li>
			<li>state: ${PlayStatus[state.status]}</li>
			<li>index: ${state.playing_track_index}</li>
			<li>volume: <input id="volume" type="range" min="0" max="65535" value="${deviceState.volume}"/></li>
			<li>Current Track: ${currentTrack.name}</li>
		</ul>
	`
}

function loginComponent() {
	return `
	<form>
		<label>
			appkey
			<input type="file" id="appkey"></input>
		</label>
		<input type="text" id="username" placeholder="username"></input>
		<input type="password" id="password" placeholder="password"></input>
		<label>
			save info
			<input type="checkbox" id="saveCheck"></input>
		</label>
		<input type="submit" value="login"></input
	</form>
	`;
}

function commandsComponent(device){
	return `
		Commands: 
		<ul>
			<li data-command="play">Play</li>
			<li data-command="pause">Pause</li>
		</ul>
	`
}

function listenCommands(){
	$('#commandsComponent').on('click', 'li', function(e){
		 switch($(this).attr('data-command')){
		 	case'play':
		 		controller.SendPlay(appState.currentIdent)
		 		break;
		 	case'pause':
		 		controller.SendPause(appState.currentIdent)
		 		break;
		 }
	})

	$("#stateComponent").on("change", '#volume', function(){
		const volume = $(this).val()
		controller.SendVolume(appState.currentIdent, volume)
	})

	$("#selectComponent").on('change', 'select', function(){
		appState.currentIdent = $(this).val()
		renderComponent('state', stateComponent())
	})

	$("#loginComponent").on("submit", "form", function(e){
		e.preventDefault()
		const reader  = new FileReader();

		reader.addEventListener("load", function(){
			const doSave = $('input[type="checkbox"]').attr('checked')
			const loginData = {
				username: $("#username").val(),
				password: $("#password").val(),
				appkey: reader.result.replace('data:;base64,','')
			}
			if (doSave) {
				chrome.storage.local.set(loginData)
			}
			doLogin(loginData)
		})

		const file = $('input[type=file]')[0].files[0];
		reader.readAsDataURL(file);
	})
}

function saveLogin(loginData) {
	chrome.storage.sync.set({
		username: username,
		password: password,
		appkey: key
	})
}

function doLogin(loginData) {
	spotcontrol.login(loginData.username, loginData.password, loginData.appkey, controller => {
		appState.loggedIn = true;
		renderAll();
		window.controller = controller
		controller.HandleUpdatesCb(handleUpdates)
	})
}

function renderComponent(section, content){
	$(`#${section}Component`).html(content);
}

function updateTrack(trackId, ident) {
	const id = spotcontrol.convert62(trackId)
	let device = getDevice(ident)
	if (device.currentTrack && device.currentTrack.id == id){
		return;
	}
	fetch(`https://api.spotify.com/v1/tracks/${id}`)
			.then(res => res.json())
			.then(data => {
				
				device.currentTrack = data
				renderComponent('state', stateComponent())
			});
}

function renderAll(){
	if (appState.loggedIn) {
		$('#loginComponent').html('')
		renderComponent('select', selectComponent())
		renderComponent('state', stateComponent())
		renderComponent('commands', commandsComponent())
	} else {
		renderComponent('login', loginComponent())
	}
}

function handleUpdates(update){
	const deviceUpdate = JSON.parse(update);

	// Volume update
	if (deviceUpdate.typ == 27) {
		let device = getCurrentDevice()
		device.device_state.volume = deviceUpdate.volume
	} else if (deviceUpdate.typ == 10) { //device notify
		let device =getDevice(deviceUpdate.ident)
		if (device) {
			Object.assign(device, deviceUpdate)
		}else{
			if(appState.devices.length == 0) {
				appState.currentIdent = deviceUpdate.ident;
			}
			device = deviceUpdate
			appState.devices.push(deviceUpdate);
			renderComponent('select', selectComponent())
		}

		if(device && device.state && device.state.track) {
			let track = device.state.track[device.state.playing_track_index]
			updateTrack(track.gid, deviceUpdate.ident)
		}
	}
	renderComponent('state', stateComponent())
}

$(document).ready(function(){
	listenCommands()
	chrome.storage.local.get(['username','password', 'appkey'], items =>{
		if(items.username && items.password) {
			doLogin(items)
		} else {
			renderAll()
		}
	})
})


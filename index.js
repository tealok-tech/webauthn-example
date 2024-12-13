document.addEventListener("DOMContentLoaded", function(event) {
	// check whether current browser supports WebAuthn
	if (!window.PublicKeyCredential) {
		alert("Error: this browser does not support WebAuthn");
		return;
	}
	const login_form = document.getElementById("login");
	login_form.addEventListener("submit", loginUser, true);
	const register_form = document.getElementById("register");
	register_form.addEventListener("submit", registerUser, true);
});

// Base64 to ArrayBuffer
function bufferDecode(value) {
	return Uint8Array.from(atob(value), c => c.charCodeAt(0));
}

// ArrayBuffer to URLBase64
function bufferEncode(value) {
	return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
		.replace(/\+/g, "-")
		.replace(/\//g, "_")
		.replace(/=/g, "");;
}

async function registerUser(e) {
	e.preventDefault();
	const username = getRegisterUsername();

	const response = await fetch('/register/begin/' + username);
	if (!response.ok) {
		throw new Error(`HTTP error on begin register: ${response.status}`);
	}
	const json = await response.json();
	var credentialCreationOptions = json;
	console.log(credentialCreationOptions);
	credentialCreationOptions.publicKey.challenge = bufferDecode(credentialCreationOptions.publicKey.challenge);
	credentialCreationOptions.publicKey.user.id = bufferDecode(credentialCreationOptions.publicKey.user.id);
	if (credentialCreationOptions.publicKey.excludeCredentials) {
		for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
			credentialCreationOptions.publicKey.excludeCredentials[i].id = bufferDecode(credentialCreationOptions.publicKey.excludeCredentials[i].id);
		}
	}

	const credential = await navigator.credentials.create({
		publicKey: credentialCreationOptions.publicKey
	});
	console.log("Credential", credential);
	let attestationObject = credential.response.attestationObject;
	let clientDataJSON = credential.response.clientDataJSON;
	let rawId = credential.rawId;

	const response2 = await fetch( '/register/finish/' + username, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({
			id: credential.id,
			rawId: bufferEncode(rawId),
			type: credential.type,
			response: {
				attestationObject: bufferEncode(attestationObject),
				clientDataJSON: bufferEncode(clientDataJSON),
			},
		})
	});
	if (!response.ok) {
		console.log(error)
		alert("failed to register " + username);
		return;
	}
	window.location.href = response2.headers.get("Location");
}

function getLoginUsername() {
	var username_element = document.querySelector("#login input[name='username']");
	return username_element.value;
}

function getRegisterUsername() {
	var username_element = document.querySelector("#register input[name='username']");
	return username_element.value;
}

async function loginUser(e) {
	e.preventDefault();
	const username = getLoginUsername();
	if (username === "") {
		alert("Please enter a username");
		return;
	}

	const response = await fetch('/login/begin/' + username);
	// 404 signals that the user doesn't exist
	if (response.status == 404) {
		showRegisterForm();
		return;
	} else if (!response.ok) {
		throw new Error(`HTTP error on begin register: ${response.status}`);
	}
	const json = await response.json();
	console.log("Login begin:", json);
	const credentialRequestOptions = json;
	credentialRequestOptions.publicKey.challenge = bufferDecode(credentialRequestOptions.publicKey.challenge);
	credentialRequestOptions.publicKey.allowCredentials.forEach(function (listItem) {
		listItem.id = bufferDecode(listItem.id)
	});

	var assertion = await navigator.credentials.get({
		publicKey: credentialRequestOptions.publicKey
	});
	console.log("Assertion", assertion);
	let authData = assertion.response.authenticatorData;
	let clientDataJSON = assertion.response.clientDataJSON;
	let rawId = assertion.rawId;
	let sig = assertion.response.signature;
	let userHandle = assertion.response.userHandle;

	const response2 = await fetch( '/login/finish/' + username, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({
			id: assertion.id,
			rawId: bufferEncode(rawId),
			type: assertion.type,
			response: {
				authenticatorData: bufferEncode(authData),
				clientDataJSON: bufferEncode(clientDataJSON),
				signature: bufferEncode(sig),
				userHandle: bufferEncode(userHandle),
			},
		})
	 });
	const json2 = await response2.json();
	console.log("Login finish", json2);
	window.location.href = response2.headers.get("Location");
}

function showRegisterForm() {
	console.log("Showing registration form");
	document.getElementById("login").style.display = "none";
	document.getElementById("register").style.display = "block";
	const login_username = getLoginUsername();
	const register_username = document.querySelector("#register input[name='username']");
	register_username.value = login_username;
}

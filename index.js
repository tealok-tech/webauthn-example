document.addEventListener("DOMContentLoaded", function(event) {
	// check whether current browser supports WebAuthn
	if (!window.PublicKeyCredential) {
		alert("Error: this browser does not support WebAuthn");
		return;
	}
	const login_form = document.getElementById("login");
	login_form.addEventListener("submit", loginUser, true);
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

function registerUser() {
	const username = document.querySelector("#email").value;
	if (username === "") {
		alert("Please enter a username");
		return;
	}

	fetch('/register/begin/' + username)
		.then(response => {
			if (!response.ok) {
				throw new Error(`HTTP error on begin register: ${response.status}`);
			}
			return response.json();
		}).then((credentialCreationOptions) => {
			console.log(credentialCreationOptions)
			credentialCreationOptions.publicKey.challenge = bufferDecode(credentialCreationOptions.publicKey.challenge);
			credentialCreationOptions.publicKey.user.id = bufferDecode(credentialCreationOptions.publicKey.user.id);
			if (credentialCreationOptions.publicKey.excludeCredentials) {
				for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
					credentialCreationOptions.publicKey.excludeCredentials[i].id = bufferDecode(credentialCreationOptions.publicKey.excludeCredentials[i].id);
				}
			}

			return navigator.credentials.create({
				publicKey: credentialCreationOptions.publicKey
			})
		})
		.then((credential) => {
			console.log(credential)
			let attestationObject = credential.response.attestationObject;
			let clientDataJSON = credential.response.clientDataJSON;
			let rawId = credential.rawId;

			fetch( '/register/finish/' + username, {
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
			})
	 })
	.then((success) => {
			alert("successfully registered " + username + "!")
		return
	})
	.catch((error) => {
		console.log(error)
		alert("failed to register " + username)
	})
}

function getLoginUsername() {
	var username_element = document.querySelector("#login input[name='username']");
	return username_element.value;
}

function loginUser(e) {
	e.preventDefault();
	const username = getLoginUsername();
	if (username === "") {
		alert("Please enter a username");
		return;
	}

	fetch('/login/begin/' + username)
		.then(response => {
			if (!response.ok) {
				throw new Error(`HTTP error on begin register: ${response.status}`);
			}
			return response.json();
		}).then((credentialRequestOptions) => {
			console.log(credentialRequestOptions)
			credentialRequestOptions.publicKey.challenge = bufferDecode(credentialRequestOptions.publicKey.challenge);
			credentialRequestOptions.publicKey.allowCredentials.forEach(function (listItem) {
				listItem.id = bufferDecode(listItem.id)
			});

			return navigator.credentials.get({
				publicKey: credentialRequestOptions.publicKey
			})
		})
		.then((assertion) => {
			console.log(assertion)
			let authData = assertion.response.authenticatorData;
			let clientDataJSON = assertion.response.clientDataJSON;
			let rawId = assertion.rawId;
			let sig = assertion.response.signature;
			let userHandle = assertion.response.userHandle;

			fetch( '/login/finish/' + username, {
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
			 })
		})
		.then((success) => {
			alert("successfully logged in " + username + "!")
			return
		})
		.catch((error) => {
			console.log(error)
			alert("failed to login " + username)
		})
}


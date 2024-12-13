package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var (
	web *webauthn.WebAuthn
	err error
)

func main() {
	err := usersDB.ReadStore()
	if err != nil {
		log.Println("Failed to read user datastore", err)
		os.Exit(1)
	}

	// Your initialization function
	web, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Foobar Corp.",          // Display Name for your site
		RPID:          "localhost",             // Generally the domain name for your site
		RPOrigin:      "http://localhost:8080", // The origin URL for WebAuthn requests
	})

	if err != nil {
		log.Fatal("failed to create WebAuthn from config:", err)
	}

	http.HandleFunc("/hello", Hello)

	http.HandleFunc("/login/begin/", BeginLogin)
	http.HandleFunc("/login/finish/", FinishLogin)

	http.HandleFunc("/logout/", Logout)

	http.HandleFunc("/register/begin/", BeginRegistration)
	http.HandleFunc("/register/finish/", FinishRegistration)

	http.Handle("/", http.FileServer(http.Dir("./")))

	serverAddress := ":8080"

	log.Println("starting server at", serverAddress)
	log.Fatal(http.ListenAndServe(serverAddress, nil))

}

// from: https://github.com/duo-labs/webauthn.io/blob/3f03b482d21476f6b9fb82b2bf1458ff61a61d41/server/response.go#L15
func jsonResponse(w http.ResponseWriter, d interface{}, c int) {
	dj, err := json.Marshal(d)
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}

func Hello(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.GetSession(r)
	var user *User
	if session == nil {
		user = UserAnonymous
	} else {
		user = session.user
	}
	t, err := template.ParseFiles("hello.tmpl")
	if err != nil {
		log.Println("Failed to load hello.tmpl", err)
		http.Error(w, "Error loading hello.tmpl", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	context := struct {
		Name string
		IsLoggedIn bool
	}{
		Name: user.Name,
		IsLoggedIn: user != UserAnonymous,
	}
	log.Println("Rendering", context)
	err = t.Execute(w, context)
	if err != nil {
		log.Println("Failed to execute hello.tmpl", err)
		http.Error(w, "Error populating hello.tmpl", http.StatusInternalServerError)
		return
	}
}

func BeginRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	// get username/friendly name
	username := strings.TrimPrefix(r.URL.Path, "/register/begin/")

	log.Println("beginning registration for: ", username)

	// get user
	user, err := usersDB.GetUser(username)
	// user doesn't exist, create new user
	if err != nil {
		displayName := strings.Split(username, "@")[0]
		user = NewUser(username, displayName)
		usersDB.PutUser(user)
	}

	registerOptions := func(credCreationOpts *protocol.PublicKeyCredentialCreationOptions) {
		credCreationOpts.CredentialExcludeList = user.CredentialExcludeList()
	}

	// generate PublicKeyCredentialCreationOptions, session data
	options, sessionData, err := web.BeginRegistration(
		user,
		registerOptions,
	)

	if err != nil {
		log.Println(err)
		jsonResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "registration",
		Value: authDb.StartSession(sessionData),
		Path:  "/",
	})

	jsonResponse(w, options, http.StatusOK)
}

func FinishRegistration(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	// get username/friendly name
	username := strings.TrimPrefix(r.URL.Path, "/register/finish/")

	log.Println("finalising registration for: ", username)

	// get user
	user, err := usersDB.GetUser(username)
	// user doesn't exist
	if err != nil {
		log.Println(err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("registration")
	if err != nil {
		log.Println("cookie:", err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionData, err := authDb.GetSession(cookie.Value)
	if err != nil {
		log.Println("cookie:", err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	credential, err := web.FinishRegistration(user, *sessionData, r)
	if err != nil {
		log.Println("finalising: ", err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	user.AddCredential(*credential)
	usersDB.PutUser(user)

	authDb.DeleteSession(cookie.Value)
	sessionStore.StartSession(w, user)
	w.Header().Set("Location", "/hello")

	jsonResponse(w, "Registration Success", http.StatusOK)
}

func BeginLogin(w http.ResponseWriter, r *http.Request) {

	// get username
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	// get username/friendly name
	username := strings.TrimPrefix(r.URL.Path, "/login/begin/")

	log.Println("user: ", username, "logging in")

	// get user
	user, err := usersDB.GetUser(username)

	// user doesn't exist
	if err != nil {
		log.Println(err)
		jsonResponse(w, err.Error(), http.StatusNotFound)
		return
	}

	// generate PublicKeyCredentialRequestOptions, session data
	options, sessionData, err := web.BeginLogin(user)
	if err != nil {
		log.Println(err)
		jsonResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "authentication",
		Value: authDb.StartSession(sessionData),
		Path:  "/",
	})

	jsonResponse(w, options, http.StatusOK)
}

func FinishLogin(w http.ResponseWriter, r *http.Request) {

	// get username
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	username := strings.TrimPrefix(r.URL.Path, "/login/finish/")

	log.Println("user: ", username, "finishing logging in")
	// get user
	user, err := usersDB.GetUser(username)

	// user doesn't exist
	if err != nil {
		log.Println(err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// load the session data
	cookie, err := r.Cookie("authentication")
	if err != nil {
		log.Println("cookie:", err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionData, err := authDb.GetSession(cookie.Value)
	if err != nil {
		log.Println("session:", err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// in an actual implementation, we should perform additional checks on
	// the returned 'credential', i.e. check 'credential.Authenticator.CloneWarning'
	// and then increment the credentials counter
	c, err := web.FinishLogin(user, *sessionData, r)
	if err != nil {
		log.Println(err)
		jsonResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	if c.Authenticator.CloneWarning {
		log.Println("cloned key detected")
		jsonResponse(w, "cloned key detected", http.StatusBadRequest)
		return
	}

	authDb.DeleteSession(cookie.Value)
	sessionStore.StartSession(w, user)

	// handle successful login
	w.Header().Set("Location", "/hello")
	jsonResponse(w, "Login Success", http.StatusOK)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	sessionStore.EndSession(w, r)
	w.Header().Set("Location", "/")
	http.Redirect(w, r, "/", http.StatusFound)
}

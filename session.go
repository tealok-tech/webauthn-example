package main

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"sync"
)

const SESSION_COOKIE_NAME = "session"

// A user's session which is used after authentication to continue to identify the user to the system.
type SessionUser struct {
	id   uuid.UUID
	user *User
}

// The store of all sessions. This is just stored in memory of the application and will therefore invalidate all sessions on process restart.
type Sessionstore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionUser
}
var sessionStore *Sessionstore = &Sessionstore{
	sessions: make(map[string]*SessionUser),
}

// Create a new store of sessions
func CreateSessionstore() Sessionstore {
	return Sessionstore{
		sessions: make(map[string]*SessionUser),
	}
}

// Get session by session UUID
func (db *Sessionstore) GetSession(r *http.Request) (*SessionUser, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	cookie, err := r.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		return nil, err
	}
	session, ok := db.sessions[cookie.Value]
	if !ok {
		return nil, fmt.Errorf("error getting session '%s': does not exist", cookie.Value)
	}
	return session, nil
}

// Start a new session for the given user and return the UUID as a string for storing in a cookie
func (db *Sessionstore) StartSession(w http.ResponseWriter, u *User) string {

	db.mu.Lock()
	defer db.mu.Unlock()

	id := uuid.New()
	db.sessions[id.String()] = &SessionUser{
		id,
		u,
	}
	log.Println("Started user session for", u.Name, id)
	http.SetCookie(w, &http.Cookie{
		Name: SESSION_COOKIE_NAME,
		Value: id.String(),
		Path: "/",
		MaxAge: 60*60*24*14,
		HttpOnly: false,
		Secure: false,
	})

	return id.String()
}

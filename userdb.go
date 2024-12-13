package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

const STORE_FILENAME = "userstore.glob"

type userdb struct {
	users map[string]*User
	mu    sync.RWMutex
}

var usersDB *userdb = &userdb{
	users: make(map[string]*User),
}

// GetUser returns a *User by the user's username
func (db *userdb) GetUser(name string) (*User, error) {

	db.mu.Lock()
	defer db.mu.Unlock()
	user, ok := db.users[name]
	if !ok {
		return &User{}, fmt.Errorf("error getting user '%s': does not exist", name)
	}

	return user, nil
}

func (d *userdb) GobDecode(b []byte) error {
	log.Println("Decoding user database")
	// Create a buffer with the input data
	buf := bytes.NewBuffer(b)

	// Create a new decoder
	dec := gob.NewDecoder(buf)

	// Create a temporary map to hold the decoded items
	var items map[string]User
	if err := dec.Decode(&items); err != nil {
		return fmt.Errorf("failed to decode items: %w", err)
	}

	// Convert the items back to pointers
	d.users = make(map[string]*User)
	for key, item := range items {
		itemCopy := item // Create a new variable to avoid pointer issues
		d.users[key] = &itemCopy
	}

	return nil
}

func (d userdb) GobEncode() ([]byte, error) {
	// Create a buffer to store the encoded data
	var buf bytes.Buffer

	// Create a new encoder
	enc := gob.NewEncoder(&buf)

	// Create a temporary map to hold the dereferenced items
	items := make(map[string]User)
	for key, ptr := range d.users {
		if ptr != nil {
			items[key] = *ptr
			log.Println("Added", ptr.Name)
		}
	}
	log.Println("Added all users")

	// Encode the temporary map
	if err := enc.Encode(items); err != nil {
		return nil, fmt.Errorf("failed to encode items: %w", err)
	}
	log.Println("Encoded", items)

	return buf.Bytes(), nil
}

// PutUser stores a new user by the user's username
func (db *userdb) PutUser(user *User) {

	db.mu.Lock()
	defer db.mu.Unlock()
	db.users[user.Name] = user
	db.writeStore()
}

func (d *userdb) ReadStore() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, err := os.Stat(STORE_FILENAME); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	f, err := os.Open(STORE_FILENAME)
	if err != nil {
		log.Println("Failed to open userstore file", err)
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	if err = dec.Decode(d); err != nil {
		log.Println("Failed to decode userstore file", err)
		return err
	}
	log.Println("Read user database from", STORE_FILENAME)
	return nil
}

func (d *userdb) writeStore() {
	f, err := os.Create(STORE_FILENAME)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(*d)
	if err != nil {
		log.Println("Failed to write store", err)
	}
}

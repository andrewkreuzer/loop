package main

import (
	"log"
	"os"
	"testing"
)

func TestNewServer(t *testing.T) {
	server := getServer()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal("No hostname")
	}
	if server.candidateID != hostname {
		t.Error("Server hostname not initialized correctly")
	}

	if server.state != follower {
		t.Error("Server state should initialize to `follower`")
	}
}

func TestUpdateState(t *testing.T) {
	getServer().updateState(candidate)
	if getServer().state != candidate {
		t.Error("Should update state to candidate")
	}
}

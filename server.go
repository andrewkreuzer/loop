package main

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

const (
	leader    = iota
	candidate = iota
	follower  = iota
)

// Server in raft cluster
type Server struct {
	mu                sync.Mutex
	candidateID       string
	state             int
	raft              *Raft
	config            *Config
	clientConnections map[string]*rpc.Client

	timer      *time.Timer
	maxTimeout int
	minTimeout int
}

var server *Server

func getServer() *Server {
	if server == nil {
		lock := &sync.Mutex{}
		lock.Lock()
		defer lock.Unlock()
		if server == nil {
			hostname, err := os.Hostname()
			if err != nil {
				log.Fatal("No hostname")
			}
			server = &Server{
				candidateID:       hostname,
				state:             follower,
				maxTimeout:        10,
				minTimeout:        2,
				clientConnections: map[string]*rpc.Client{},
			}
		}
	}

	return server
}

func (server *Server) start() {
	server.config = LoadConfig()
	server.raft = NewRaft(server.config.Nodes)

	rpc.Register(server.raft)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":8080")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

	go server.startTimer()

	http.HandleFunc("/", handleLogRequest)
	go http.ListenAndServe(":80", nil)

	// TODO: remove runtime loop??
	for {
		switch server.state {
		case leader:
			server.runAsLeader()
		case candidate:
			server.runAsCandidate()
		case follower:
			server.runAsFollower()
		}
	}
}

// LogRequest json object for requesting a new log entry
type LogRequest struct {
	Command string `json:"command"`
}

func handleLogRequest(w http.ResponseWriter, r *http.Request) {
	if server.raft.VotedFor == "" {
		w.Write([]byte("No leader ellected yet\n"))
		return
	}

	if server.raft.VotedFor != server.candidateID {
		leader, err := server.config.getNode(server.raft.VotedFor)
		if err != nil {
			log.Println(err)
		}
		http.Redirect(w, r, leader.URL, 301)
	}

	if server.state == leader {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
		}

		var logRequest LogRequest
		json.Unmarshal(body, &logRequest)
		// TODO: validate the command in some way
		server.raft.appendRequest(logRequest.Command)
	}
}

func (server *Server) runAsLeader() {
	log.Println("Running as leader on term:", server.raft.CurrentTerm)

	server.stopTimer()
	server.sendAppendEntries()
	time.Sleep(1 * time.Second)
}

func (server *Server) runAsCandidate() {
	log.Println("Running as candidate on term:", server.raft.CurrentTerm)

	server.raft.updateTerm(server.raft.CurrentTerm + 1)
	server.connectClients()
	server.sendRequestVote()
	time.Sleep(1 * time.Second)
}

func (server *Server) runAsFollower() {
	log.Println("Running as follower on term:", server.raft.CurrentTerm)
	if len(server.raft.Log) >= 1 {
		logEntry := server.raft.Log[len(server.raft.Log)-1]
		log.Println("Log lenth:", len(server.raft.Log), "Log Command:", logEntry.Command, "Log Term:", logEntry.Term, "CurrentTerm:", server.raft.CurrentTerm)
	}
	time.Sleep(1 * time.Second)
}

func (server *Server) updateState(state int) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.state = state
}

func (server *Server) startTimer() {
	f := func() {
		server.updateState(candidate)
	}
	rand.Seed(time.Now().UnixNano())
	timeoutLength := rand.Intn(server.maxTimeout-server.minTimeout) + server.minTimeout
	timeoutDuration := time.Duration(timeoutLength * int(time.Second))
	server.timer = time.AfterFunc(timeoutDuration, f)
}

func (server *Server) stopTimer() {
	if server.timer != nil {
		server.timer.Stop()
	}
}

func (server *Server) resetTimer() {
	if server.timer != nil {
		server.timer.Stop()
		timeoutLength := rand.Intn(server.maxTimeout-server.minTimeout) + server.minTimeout
		timeoutDuration := time.Duration(timeoutLength * int(time.Second))
		server.timer.Reset(timeoutDuration)
	}
}

func (server *Server) sendAppendEntries() {
	for _, client := range server.config.Nodes {
		nextIndex := server.raft.NextIndex[client.CandidateID]
		prevIndex := nextIndex - 1
		lastIndex := len(server.raft.Log)

		args := &AppendEntriesArgs{
			Term:         server.raft.CurrentTerm,
			LeaderID:     server.candidateID,
			PrevLogIdx:   prevIndex,
			PrevLogTerm:  server.raft.getLogTerm(prevIndex),
			Entries:      server.raft.Log[prevIndex:lastIndex],
			LeaderCommit: server.raft.CommitedIndex,
		}

		conn := server.clientConnections[client.CandidateID]
		appendEntriesReply := new(AppendEntriesReply)
		err := conn.Call("Raft.AppendEntries", args, appendEntriesReply)
		if err != nil {
			log.Println("Error calling AppendEntries on:", client.CandidateID, err)
			// TODO: handle closed connections and our list of nodes
			continue
		}

		if appendEntriesReply.Term > server.raft.CurrentTerm {
			server.raft.updateTerm(appendEntriesReply.Term)
			server.updateState(follower)
			return
		}

		if appendEntriesReply.Success {
			server.raft.NextIndex[client.CandidateID] = lastIndex + 1
			server.raft.MatchIndex[client.CandidateID] = lastIndex
		} else {
			server.raft.NextIndex[client.CandidateID] = server.raft.NextIndex[client.CandidateID] - 1
		}
	}
}

func (server *Server) sendRequestVote() {
	server.raft.updateVotedFor(server.candidateID)

	votesRecieved := float32(1)
	for _, client := range server.config.Nodes {
		lastIndex := len(server.raft.Log) - 1
		if lastIndex < 0 {
			lastIndex = 0
		}

		args := &RequestVoteArgs{
			Term:         server.raft.CurrentTerm,
			CandidateID:  server.candidateID,
			LastLogTerm:  server.raft.getLogTerm(lastIndex),
			LastLogIndex: lastIndex,
		}

		conn := server.clientConnections[client.CandidateID]
		requestVoteReply := new(RequestVoteReply)
		err := conn.Call("Raft.RequestVote", args, requestVoteReply)
		if err != nil {
			log.Println(err)
		}

		if requestVoteReply.VoteGranted {
			log.Println("Vote recieved:", client.CandidateID)
			votesRecieved++
		}

		if requestVoteReply.Term > server.raft.CurrentTerm {
			server.raft.updateTerm(requestVoteReply.Term)
			server.updateState(follower)
			return
		}
	}

	if votesRecieved >= float32(len(server.config.Nodes))/2 {
		log.Println("Reached leader status")
		server.updateState(leader)
	}
}

func (server *Server) connectClient(node NodeInfo) *rpc.Client {
	var err error
	client, err := rpc.DialHTTP("tcp", node.Endpoint)
	if err != nil {
		log.Println("Error dialing:", err)
	}

	return client
}

func (server *Server) connectClients() {
	for _, client := range server.config.Nodes {
		server.clientConnections[client.CandidateID] = server.connectClient(client)
	}
}

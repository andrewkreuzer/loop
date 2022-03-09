package main

import (
	"errors"
	"log"
	"math"
	"sync"
)

// Raft State object
type Raft struct {
	// Mutex
	mu *sync.Mutex
	// last term server has seen
	CurrentTerm int

	// candidateid id which server voted for in current term
	VotedFor string

	// log entries
	Log []RaftLog

	// highest index of log entry known to be commited
	CommitedIndex int

	// highest index of log entry applied to state machine
	LastApplied int

	// next log entry to send to each server leader index + 1
	NextIndex map[string]int

	// highest log entry index known to be replicated from each server
	MatchIndex map[string]int
}

func initializeIndexs(nodes []NodeInfo, fill int) map[string]int {
	indexMap := make(map[string]int)
	for _, node := range nodes {
		indexMap[node.CandidateID] = fill
	}

	return indexMap
}

// NewRaft create the raft stucture
func NewRaft(nodes []NodeInfo) *Raft {
	return &Raft{
		mu:            &sync.Mutex{},
		CurrentTerm:   0,
		VotedFor:      "",
		Log:           make([]RaftLog, 0),
		CommitedIndex: 0,
		LastApplied:   0,
		NextIndex:     initializeIndexs(nodes, 1),
		MatchIndex:    initializeIndexs(nodes, 0),
	}
}

// RaftLog entry
type RaftLog struct {
	// command to envoke on state machine
	Command interface{}

	// term of log entry recieved by leader
	Term int
}

// AppendEntriesArgs rpc inputs
type AppendEntriesArgs struct {
	Term         int
	LeaderID     string
	PrevLogIdx   int
	PrevLogTerm  int
	Entries      []RaftLog
	LeaderCommit int
}

// AppendEntriesReply rpc outputs
type AppendEntriesReply struct {
	Term    int
	Success bool
}

// AppendEntries rpc function
func (raft *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	server.resetTimer()

	if raft.VotedFor != args.LeaderID {
		return errors.New("VotedFor is not LeaderID")
	}

	if args.Term < raft.CurrentTerm {
		reply.Term = raft.CurrentTerm
		reply.Success = false
		return errors.New("Server term greater than request term")
	}

	if len(raft.Log) >= 1 {
		if args.PrevLogTerm != raft.getLogTerm(args.PrevLogIdx) {
			reply.Term = raft.CurrentTerm
			reply.Success = false
			return errors.New("Server previous log term differs")
		}
	}

	raft.appendToLog(args.PrevLogIdx, args.Entries)

	if args.LeaderCommit > raft.CommitedIndex {
		raft.updateCommitedIndex(int(math.Min(
			float64(args.LeaderCommit),
			float64(len(raft.Log)-1),
		)))
	}

	reply.Term = raft.CurrentTerm
	reply.Success = true

	return nil
}

// RequestVoteArgs rpc inputs
type RequestVoteArgs struct {
	Term         int
	CandidateID  string
	LastLogTerm  int
	LastLogIndex int
}

// RequestVoteReply rpc outputs
type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

// RequestVote rpc function
func (raft *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	server.resetTimer()

	if args.Term < raft.CurrentTerm {
		reply.Term = raft.CurrentTerm
		reply.VoteGranted = false
		return errors.New("Requested vote term less than currentTerm")
	}

	if raft.VotedFor == "" || args.CandidateID == raft.VotedFor {
		if args.LastLogIndex >= raft.CommitedIndex && args.LastLogTerm >= raft.CurrentTerm {
			log.Println("Candidate meets criteria:", args.CandidateID)
			raft.updateVotedFor(args.CandidateID)
			reply.Term = raft.CurrentTerm
			reply.VoteGranted = true
		}
	}

	if args.Term > raft.CurrentTerm {
		log.Println("Requested vote term greater than currentTerm")
		raft.updateTerm(args.Term)
		server.updateState(follower)
	}

	return nil
}

func (raft *Raft) appendToLog(prevLogIdx int, entries []RaftLog) {
	raft.mu.Lock()
	defer raft.mu.Unlock()

	j := prevLogIdx
	k := len(raft.Log)
	for j < k {
		if raft.Log[j].Term != entries[j-prevLogIdx].Term {
			raft.Log = raft.Log[:j]
			raft.Log = append(raft.Log, entries[j-prevLogIdx:]...)
			return
		}

		j++
	}

	raft.Log = append(raft.Log, entries...)
}

func (raft *Raft) getLogTerm(index int) int {
	if index < 1 || index > len(raft.Log) {
		return 0
	}

	return raft.Log[index-1].Term
}

// TODO: make interface usable from http server
func (raft *Raft) appendRequest(command interface{}) error {
	log.Println("Adding log entry:", len(raft.Log)+1, command)
	raft.mu.Lock()
	defer raft.mu.Unlock()
	newLog := RaftLog{Command: command, Term: raft.CurrentTerm}
	raft.Log = append(raft.Log, newLog)

	return nil
}

func (raft *Raft) updateCommitedIndex(idx int) {
	raft.mu.Lock()
	defer raft.mu.Unlock()
	raft.CommitedIndex = idx
}

func (raft *Raft) updateTerm(term int) error {
	log.Println("updating to term:", term)
	raft.mu.Lock()
	defer raft.mu.Unlock()
	raft.CurrentTerm = term

	return nil
}

func (raft *Raft) updateVotedFor(candidateID string) {
	raft.mu.Lock()
	defer raft.mu.Unlock()

	raft.VotedFor = candidateID
}

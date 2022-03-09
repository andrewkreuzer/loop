package main

import "testing"

func TestNewRaft(t *testing.T) {
	nodes := []NodeInfo{
		{CandidateID: "test1", Endpoint: "test1:8080", URL: "http://test1:80"},
		{CandidateID: "test2", Endpoint: "test2:8080", URL: "http://test2:80"},
		{CandidateID: "test3", Endpoint: "test3:8080", URL: "http://test3:80"},
	}
	raft := NewRaft(nodes)

	if raft.CurrentTerm != 0 {
		t.Error("CurrentTerm should initialize to 0", raft.CurrentTerm)
	}

	if raft.VotedFor != "" {
		t.Error("VotedFor should initialize to an emtpy string:", raft.VotedFor)
	}

	if raft.CommitedIndex != 0 {
		t.Error("CommitedIndex should initialize to 0", raft.CommitedIndex)
	}

	if raft.LastApplied != 0 {
		t.Error("LastApplied should initialize to 0:", raft.LastApplied)
	}

	for _, node := range nodes {
		if raft.NextIndex[node.CandidateID] != 1 {
			t.Error("NextIndex should initialize to 1 for all node:", node.CandidateID, "value:", raft.NextIndex[node.CandidateID])
		}

		if raft.MatchIndex[node.CandidateID] != 0 {
			t.Error("MatchIndex should initialize to 0 for all node:", node.CandidateID, "value:", raft.NextIndex[node.CandidateID])
		}
	}
}

func TestAppendToLog(t *testing.T) {
	nodes := []NodeInfo{
		{CandidateID: "test1", Endpoint: "test1:8080", URL: "http://test1:80"},
		{CandidateID: "test2", Endpoint: "test2:8080", URL: "http://test2:80"},
		{CandidateID: "test3", Endpoint: "test3:8080", URL: "http://test3:80"},
	}
	raft := NewRaft(nodes)

	entries := []RaftLog{
		{Command: nil, Term: 1},
		{Command: "test", Term: 1},
		{Command: "test", Term: 1},
	}
	raft.appendToLog(0, entries)

	if len(raft.Log) != 3 {
		t.Error("Should add entries:", entries, "got:", raft.Log)
	}

	entries = []RaftLog{
		{Command: nil, Term: 1},
		{Command: "test", Term: 2},
		{Command: "test", Term: 3},
	}
	raft.appendToLog(2, entries)

	if raft.Log[1].Term != 1 && raft.Log[3].Term != 2 {
		t.Error("Should overwrite the log from the 2nd index", raft.Log)
	}

	entries = []RaftLog{
		{Command: nil, Term: 3},
		{Command: "test", Term: 3},
		{Command: "test", Term: 3},
	}
	raft.appendToLog(len(raft.Log), entries)

	lastLogIdx := len(raft.Log) - 1
	if raft.Log[lastLogIdx].Term != 3 ||
		raft.Log[lastLogIdx-1].Term != 3 ||
		raft.Log[lastLogIdx-2].Term != 3 ||
		raft.Log[lastLogIdx-4].Term != 2 {
		t.Error("Should append to the end of the log", raft.Log)
	}
}

func TestGetLogTerm(t *testing.T) {
	nodes := []NodeInfo{
		{CandidateID: "test1", Endpoint: "test1:8080", URL: "http://test1:80"},
		{CandidateID: "test2", Endpoint: "test2:8080", URL: "http://test2:80"},
		{CandidateID: "test3", Endpoint: "test3:8080", URL: "http://test3:80"},
	}
	raft := NewRaft(nodes)

	entries := []RaftLog{
		{Command: nil, Term: 1},
		{Command: "test", Term: 1},
		{Command: "test", Term: 1},
	}
	raft.appendToLog(0, entries)

	if raft.getLogTerm(1) != 1 {
		t.Error("Should get term from entry 1 (index=0) and should equal 1")
	}

	if raft.getLogTerm(0) != 0 {
		t.Error("Should return 0 as entry 0 is less than 1")
	}

	if raft.getLogTerm(50) != 0 {
		t.Error("Should return 0 as entry 50 is greater than log length")
	}
}

func TestUpdateCommitedIndex(t *testing.T) {
	nodes := []NodeInfo{
		{CandidateID: "test1", Endpoint: "test1:8080", URL: "http://test1:80"},
		{CandidateID: "test2", Endpoint: "test2:8080", URL: "http://test2:80"},
		{CandidateID: "test3", Endpoint: "test3:8080", URL: "http://test3:80"},
	}
	raft := NewRaft(nodes)

	raft.updateCommitedIndex(1)
	if raft.CommitedIndex != 1 {
		t.Error("Should update CommitedIndex to 1, got:", raft.CommitedIndex)
	}
}

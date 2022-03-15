package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/rpc"
	"sync"
)

// NodeInfo for node config
type NodeInfo struct {
	CandidateID string `json:"candidateID"`
	Endpoint    string `json:"endpoint"`
	URL         string `json:"url"`

	Client *rpc.Client
}

// Config for nodes in cluster
type Config struct {
	mu    sync.Mutex
	Nodes []NodeInfo `json:"nodes"`
}

// LoadConfig reads config file and returns Config object
func LoadConfig() *Config {
	config := new(Config)
	config.mu.Lock()
	defer config.mu.Unlock()
	content, err := ioutil.ReadFile("./cluster_config.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	// remove this server from our list of nodes
	config.removeNode(server.candidateID)

	return config
}

func (config *Config) getNode(candidateID string) (*NodeInfo, error) {
	for _, node := range config.Nodes {
		if node.CandidateID == candidateID {
			return &node, nil
		}
	}

	return nil, errors.New("Node not found")
}

func (config *Config) removeNode(candidateID string) error {
	for i, node := range config.Nodes {
		if node.CandidateID == server.candidateID {
			config.Nodes[i] = config.Nodes[len(config.Nodes)-1]
			config.Nodes = config.Nodes[:len(config.Nodes)-1]
		}
	}

	return nil
}

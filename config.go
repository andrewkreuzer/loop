package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

// TODO: is it neccessary to recreate the connection every time???
func (config *Config) connectClient(candidateID string) (*rpc.Client, error) {
	for _, client := range config.Nodes {
		if candidateID == client.CandidateID {
			var err error
			client.Client, err = rpc.DialHTTP("tcp", client.Endpoint)
			if err != nil {
				log.Println("Error dialing:", err)
			}

			return client.Client, nil
		}
	}

	return nil, fmt.Errorf("Unable to connect to: %s", candidateID)
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

// func (config *Config) connectClients() {
// 	clientCount := 0
// 	for _, client := range server.config.Nodes {
// 		if client.CandidateID == server.candidateID {
// 			continue
// 		}

// 		log.Println("Connecting to", client.CandidateID)

// 		if client.Client == nil {
// 			var err error
// 			client.Client, err = rpc.DialHTTP("tcp", client.Endpoint)
// 			if err != nil {
// 				log.Println("Error dialing:", err)
// 			}
// 			clientCount++
// 		}
// 	}

// 	if clientCount < 2 {
// 		log.Fatalln("Cannot run a cluster with less than 3 nodes")
// 	}

// }

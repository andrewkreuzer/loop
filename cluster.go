package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

// Cluster ...
type Cluster struct {
	Nodes   []NodeInfo
	Joining []NodeInfo
}

func (cluster *Cluster) initIntoCluster() {
	log := getLogger()
	self := getSelfInfo()

	var nodes []NodeInfo
	getNodesURL := fmt.Sprint(cluster.Nodes[0].URL, "/heartbeat")
	resp, err := http.Post(getNodesURL, "application/json", nil)
	if err != nil {
		log.Error("Heartbeat failed:", err)
	} else {
		decoder := json.NewDecoder(resp.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&nodes)
		if err != nil {
			log.Error("Error: ", err)
		}
	}

	data, err := json.Marshal(self)
	if err != nil {
		log.Error("Error marshaling selfNodeInfo:", err)
	}
	for _, node := range nodes {
		if node.Name == self.Name {
			continue
		}

		addURL := fmt.Sprint(node.URL, "/join")
		_, err := http.Post(addURL, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Error(err)
		}

		log.Info("Sent join requests to:", cluster.Nodes)
	}
}

func (cluster *Cluster) add(nodeInfo NodeInfo) bool {
	log := getLogger()

	cluster.Joining = append(cluster.Joining, nodeInfo)
	go cluster.runConsensus()

	log.Info("Node added to joinList:", nodeInfo.Name)
	return true
}

func (cluster *Cluster) update(nodes []NodeInfo) {
	log := getLogger()

	log.Info("updating cluster")
OUTER:
	for _, node := range nodes {
		for _, cnode := range cluster.Nodes {
			if cnode.Name == node.Name {
				continue OUTER
			}
		}

		cluster.Nodes = append(cluster.Nodes, node)
	}
}

// Query ...
type Query struct {
	NodeName string `json:"nodeName"`
}

func (cluster *Cluster) query(query Query) bool {
	if query.NodeName != "" {

		for _, node := range cluster.Joining {
			if query.NodeName == node.Name {
				return true
			}
		}

		for _, node := range cluster.Nodes {
			if query.NodeName == node.Name {
				return true
			}
		}
	}

	return false
}

func (cluster *Cluster) sendQuery(query Query) int {
	log := getLogger()
	self := getSelfInfo()

	nodesAckknowledged := 0
	for _, node := range cluster.Nodes {
		if node.Name == self.Name {
			continue
		}

		queryURL := fmt.Sprint(node.URL, "/query")
		jsonData, err := json.Marshal(query)
		if err != nil {
			log.Error("Failed marshaling data list:", err)
		}

		resp, err := http.Post(queryURL, "json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Error("Query failed:", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Error("Error reading response body")
		}

		if string(body) != "true" {
			log.Error("Node", node.Name, "does not know:", query.NodeName)
		}

		log.Info("Node exists:", node.Name)
		nodesAckknowledged++
	}

	return nodesAckknowledged
}

func (cluster *Cluster) runConsensus() {
	log := getLogger()

	for _, node := range cluster.Joining {
		query := Query{NodeName: node.Name}
		nodesAckknowledged := cluster.sendQuery(query)

		if nodesAckknowledged > len(cluster.Nodes)/2 {
			log.Info("Reached consensus on node:", node.Name)
			node.Alive = true
			cluster.update([]NodeInfo{node})

			for _, cnode := range cluster.Nodes {
				cnode.sendMessage(cluster.Nodes)
			}
		}
	}

	cluster.Joining = nil
}

// NodeInfo ...
type NodeInfo struct {
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	URL       string    `json:"url"`
	Key       string    `json:"key"`
	Alive     bool      `json:"alive"`
	CreatedAt time.Time `json:"createAt"`
}

func (nodeInfo *NodeInfo) sendMessage(message interface{}) {
	log := getLogger()
	messageURL := fmt.Sprint(nodeInfo.URL, "/message")
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Error("Failed marshaling data list:", err)
	}

	resp, err := http.Post(messageURL, "json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("Query failed:", err)
	}

	_, err = io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Error("Error reading response body")
	}
}

var selfNodeInfo *NodeInfo

func getSelfInfo() *NodeInfo {
	log := getLogger()

	if selfNodeInfo == nil {
		lock.Lock()
		defer lock.Unlock()
		if selfNodeInfo == nil {
			hostname, err := os.Hostname()
			if err != nil {
				log.Error("Hostname error:", hostname)
			}
			url := fmt.Sprint("http://", hostname, ":", os.Getenv("PORT"))

			selfNodeInfo = &NodeInfo{
				Name:      hostname,
				Location:  os.Getenv("LOCATION"),
				URL:       url,
				Key:       "random",
				Alive:     false,
				CreatedAt: time.Now(),
			}
		}
	}

	return selfNodeInfo
}

var cluster *Cluster
var lock = &sync.Mutex{}

func getCluster() *Cluster {
	log := getLogger()

	if cluster == nil {
		lock.Lock()
		defer lock.Unlock()

		if cluster == nil {
			nodesFile, err := ioutil.ReadFile(os.Getenv("NODES_FILE"))
			if err != nil {
				log.Fatal("Error when opening file: ", err)
			}

			var payload []NodeInfo
			err = json.Unmarshal(nodesFile, &payload)
			if err != nil {
				log.Fatal("Error during Unmarshal(): ", err)
			}

			cluster = &Cluster{Nodes: payload, Joining: make([]NodeInfo, 0)}
		}
	}

	return cluster
}

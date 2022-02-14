package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Heartbeat ...
func Heartbeat() {
	cluster := getCluster()
	selfNodeInfo := getSelfInfo()
	log := getLogger()

	// dumby data for now
	data := [...]string{"thing", "thing2"}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error("Failed marshaling data list:", err)
	}

	for {
		log.Info("Running heartbeats")
		for _, node := range cluster.Nodes {
			if node.Name == selfNodeInfo.Name || !node.Alive {
				continue
			}

			hearbeatURL := fmt.Sprint(node.URL, "/heartbeat")

			resp, err := http.Post(hearbeatURL, "json", bytes.NewBuffer(jsonData))
			if err != nil {

				log.Error("Heartbeat failed:", err)

			} else {
				body, _ := io.ReadAll(resp.Body)
				defer resp.Body.Close()

				if string(body) != "Ack" {
					log.Error("Node did not acknowledge:", node.Name)
				}

				log.Info("Heartbeat success for:", node.Name)
			}

		}
		time.Sleep(2 * time.Second)
	}
}

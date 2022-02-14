package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

func heartbeat(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Ack"))
}

func clusterNodes(w http.ResponseWriter, req *http.Request) {
	cluster := getCluster()
	log := getLogger()
	nodes, err := json.Marshal(cluster.Nodes)
	if err != nil {
		log.Error("Failed marshaling cluster list")
	}
	w.Write(nodes)
}

func query(w http.ResponseWriter, req *http.Request) {
	log := getLogger()
	cluster := getCluster()

	var query Query
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&query)
	if err != nil {
		log.Error("Error: ", err)
	}

	nodeExists := strconv.FormatBool(cluster.query(query))
	w.Write([]byte(nodeExists))
}

func join(w http.ResponseWriter, req *http.Request) {
	cluster := getCluster()
	log := getLogger()

	if req.Header.Get("Content-Type") != "application/json" {
		w.Write([]byte("When adding you must send json"))
		log.Warn("message did not include json")
		return
	}

	var n NodeInfo

	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&n)
	if err != nil {
		log.Error("Error: ", err)
	}

	for _, node := range cluster.Nodes {
		if n.Name == node.Name {
			log.Info("Node already in cluster:", n.Name)
			return
		}
	}

	ok := cluster.add(n)
	if !ok {
		errorResponse(w, "Error joining", 500)
		return
	}

	w.Write([]byte("Added to joining"))
}

func recieveMessage(w http.ResponseWriter, req *http.Request) {
	cluster := getCluster()
	log := getLogger()

	var nodes []NodeInfo
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&nodes)
	if err != nil {
		log.Error("Error: ", err)
	}

	cluster.update(nodes)
}

func main() {
	log := NewLogger("loop_log.txt", true)
	defer log.logFile.Close()

	endpoint := fmt.Sprint(os.Getenv("URL"), ":", os.Getenv("PORT"))

	http.HandleFunc("/join", join)
	http.HandleFunc("/query", query)
	http.HandleFunc("/getNodes", clusterNodes)
	http.HandleFunc("/heartbeat", heartbeat)
	http.HandleFunc("/message", recieveMessage)
	log.Info("Starting Server at:", endpoint)
	go func() {
		log.Warn(http.ListenAndServe(endpoint, nil))
	}()

	getCluster().initIntoCluster()

	go Heartbeat()

	select {}
}

func errorResponse(w http.ResponseWriter, message string, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
}

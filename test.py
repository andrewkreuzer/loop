#!/usr/bin/env python3
import requests
import json

with open("./examples/join_example.json") as f:
    node_data = json.loads(f.read())

r = requests.post("http://localhost:8081/join", json=node_data)
print("Join: ", r.text)

with open("./examples/query_example.json") as f:
    query_data = json.loads(f.read())

r = requests.post("http://localhost:8081/query", json=query_data)
print("Query: ", r.text)

r = requests.get("http://localhost:8081/nodeList")
print("NodeList: ", r.text)

r = requests.get("http://localhost:8081/heartbeat")
print("Heartbeat: ", r.text)

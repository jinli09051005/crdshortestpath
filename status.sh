#!/bin/bash

make deploy IMG=jinli.harbor.com/jinlik8s-crd/jinli-dijkstra-crd:v1.0.2
kubectl -ncrdshortestpath-system get pods -w

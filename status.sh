#!/bin/bash

make deploy IMG=jinli.harbor.com/jinlik8s-crd/jinli-dijkstra-crd-controller:v1.0.1
kubectl -ncrdshortestpath-system get pods -w

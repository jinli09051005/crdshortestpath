#!/bin/bash



make undeploy

nerdctl -n k8s.io rmi -f $(nerdctl -n k8s.io images | grep di | awk '{print $3}')

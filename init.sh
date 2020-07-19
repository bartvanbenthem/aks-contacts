#!/bin/bash

# initialize go modules
go mod init github.com/bartvanbenthem/aks2contact
# get the correct go-client module
go get k8s.io/client-go@kubernetes-1.16.7


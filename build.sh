#!/bin/bash

export GOPATH=`pwd`
export GOBIN=`pwd`/bin

go get -d ungoliant
go install ungoliant

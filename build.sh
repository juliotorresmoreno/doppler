#!/bin/sh

app_name=doppler

mkdir -p bin
GOOS=windows GOARCH=amd64 go build -o bin/$app_name.exe

GOOS=linux GOARCH=amd64 go build -o bin/$app_name

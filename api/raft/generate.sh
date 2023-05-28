#!/bin/bash

~/go/bin/oapi-codegen -generate gin,spec,types --package raft raft.yaml > server/server.gen.go
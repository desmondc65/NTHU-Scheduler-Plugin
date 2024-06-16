#!/bin/bash

if [ -z "$1" ]
then
    echo "no file specified"
    exit 1
else
    kubectl create -f "$1"
fi
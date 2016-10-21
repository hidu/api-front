#!/bin/bash
cd $(dirname $0)
gox -osarch linux/amd64

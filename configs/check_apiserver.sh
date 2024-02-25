#!/bin/sh

errorExit() {
    echo "*** $*" 1>&2
    exit 1
}

curl --silent --max-time 2 --insecure http://localhost:8080/health -o /dev/null || errorExit "Error GET http://localhost:8080/health"
if ip addr | grep -q 10.0.1.254; then
    curl --silent --max-time 2 --insecure http://10.0.1.254:8080/health -o /dev/null || errorExit "Error GET http://10.0.1.254:8080/health"
fi
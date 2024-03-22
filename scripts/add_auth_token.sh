#!/bin/bash

server_exec() {
    ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

ACCESS_TOKEN="$(cat ~/.git_token_loader)"

server_exec $1 'echo -en "\n\n" | ssh-keygen -t rsa'
server_exec $1 'ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts'
server_exec $1 'curl -H "Authorization: token '"$ACCESS_TOKEN"'" --data "{\"title\":\"'"key:\$(hostname)"'\",\"key\":\"'"\$(cat ~/.ssh/id_rsa.pub)"'\"}" https://api.github.com/user/keys'

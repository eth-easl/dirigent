//
// Created by lcvetkovic on 5/10/23.
//

#ifndef CLUSTER_MANAGER_ENDPOINT_H
#define CLUSTER_MANAGER_ENDPOINT_H

#include <arpa/inet.h>
#include <limits.h>
#include <sys/socket.h>
#include <unistd.h>

#include <iostream>
#include <string>
#include <vector>

using namespace std;

namespace master_node::ingress {

    constexpr uint SERVERLESS_RX_PORT = 9090;


    void serve();

    void create_thread_pool();


}

#endif //CLUSTER_MANAGER_ENDPOINT_H

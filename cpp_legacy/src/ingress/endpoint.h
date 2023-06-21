//
// Created by lcvetkovic on 5/10/23.
//

#ifndef CLUSTER_MANAGER_ENDPOINT_H
#define CLUSTER_MANAGER_ENDPOINT_H

#include <arpa/inet.h>
#include <sys/socket.h>

#include <climits>
#include <iostream>
#include <string>
#include <vector>

#include <spdlog/spdlog.h>

#include "request_buffer.h"
#include "thread_pool.h"

using namespace std;

namespace ingress {

    constexpr uint SERVERLESS_RX_PORT = 9090;
    constexpr int CONNECTION_THREAD_POOL_SIZE = 1;

    class Ingress {

    public:
        Ingress() : threadPool(connection_thread_pool(CONNECTION_THREAD_POOL_SIZE)) {}

        [[noreturn]]
        void start_serving();

    private:
        const connection_thread_pool threadPool;

    };


}

#endif //CLUSTER_MANAGER_ENDPOINT_H

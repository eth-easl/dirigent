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

#include "request_buffer.h"

using namespace std;

namespace ingress {

    constexpr uint SERVERLESS_RX_PORT = 9090;

    class Ingress {

    public:
        Ingress(const RequestBuffer<byte> &requestBuffer) : requestBuffer(requestBuffer) {}

        void startServing();

    private:
        const RequestBuffer<byte> &requestBuffer;

        void create_thread_pool();

    };


}

#endif //CLUSTER_MANAGER_ENDPOINT_H

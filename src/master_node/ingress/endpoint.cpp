//
// Created by lcvetkovic on 5/10/23.
//

#include "endpoint.h"

void master_node::ingress::serve() {
    int ingress_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (ingress_fd < 0)
        throw runtime_error("Could not create a socket.");

    int opt_value = 1;
    if (setsockopt(ingress_fd, SOL_SOCKET, SO_REUSEADDR | SO_REUSEPORT, &opt_value, sizeof(opt_value)) != 0)
        throw new runtime_error("Could not set socket options.");

    struct sockaddr_in socket_address;
    socket_address.sin_family = AF_INET;
    socket_address.sin_addr.s_addr = INADDR_ANY;
    socket_address.sin_port = htons(SERVERLESS_RX_PORT);
    constexpr int socket_address_length = sizeof(socket_address);

    if (bind(ingress_fd, (struct sockaddr *) &socket_address, sizeof(socket_address)) < 0)
        throw runtime_error("Error binding to a socket.");

    int max_connections_allowed = SOMAXCONN;
    if (listen(ingress_fd, max_connections_allowed) != 0)
        throw runtime_error("Error while listening on a socket.");

    while (true) {
        int rx_fd = accept(ingress_fd, (struct sockaddr *) &socket_address, (socklen_t *) &socket_address_length);

        vector<char *> datagram;
        constexpr int BUFFER_SIZE = 4;
        char *buf = new char[BUFFER_SIZE];
        // TODO(lazar): remember to deallocate this memory somewhere

        int bytes_read;
        while ((bytes_read = read(rx_fd, buf, BUFFER_SIZE)) != 0) {
            datagram.push_back(buf);
            cout << string(buf) << endl;
            buf = new char[BUFFER_SIZE];
        }

        // TODO(lazar): offload work to a worker thread

        close(rx_fd);
        break;
    }

    shutdown(ingress_fd, SHUT_RDWR);
}

void master_node::ingress::create_thread_pool() {
    // TODO(lazar): thread pool of connection handlers
}

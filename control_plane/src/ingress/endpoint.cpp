//
// Created by lcvetkovic on 5/10/23.
//

#include "endpoint.h"

void ingress::Ingress::start_serving() {
    int ingress_fd = socket(AF_INET, SOCK_STREAM, 0);
    if (ingress_fd < 0)
        throw runtime_error("Could not create a socket.");

    int opt_value = 1;
    if (setsockopt(ingress_fd, SOL_SOCKET, SO_REUSEADDR | SO_REUSEPORT, &opt_value, sizeof(opt_value)) != 0)
        throw runtime_error("Could not set socket options.");

    struct sockaddr_in socket_address = {};
    socket_address.sin_family = AF_INET;
    socket_address.sin_addr.s_addr = INADDR_ANY;
    socket_address.sin_port = htons(SERVERLESS_RX_PORT);
    constexpr int socket_address_length = sizeof(socket_address);

    if (bind(ingress_fd, (struct sockaddr *) &socket_address, sizeof(socket_address)) < 0)
        throw runtime_error("Error binding to a socket.");

    int max_connections_allowed = SOMAXCONN;
    if (listen(ingress_fd, max_connections_allowed) != 0)
        throw runtime_error("Error while listening on a socket.");

    spdlog::info("Ingress running on {}", SERVERLESS_RX_PORT);

    int conn_pool_lb = 0;
    while (true) {
        int tx_fd = accept(ingress_fd, (struct sockaddr *) &socket_address, (socklen_t *) &socket_address_length);

        // conn_metadata object deleted once request processing finished
        auto *conn_metadata = new client_conn_desc(tx_fd);
        threadPool.send_message(conn_pool_lb, conn_metadata);
        conn_pool_lb = (conn_pool_lb + 1) % threadPool.size();
    }

    shutdown(ingress_fd, SHUT_RDWR);
}

//
// Created by lcvetkovic on 5/16/23.
//

#ifndef CLUSTER_MANAGER_THREAD_POOL_H
#define CLUSTER_MANAGER_THREAD_POOL_H

#include <unistd.h>

#include <condition_variable>
#include <mutex>
#include <queue>
#include <thread>
#include <vector>

#include <spdlog/spdlog.h>

#include "ingress_semaphore.h"

using namespace std;

namespace ingress {

    struct client_conn_desc {
        const int tx_fd;
        //vector<char *> request;

        explicit client_conn_desc(int tx_fd) : tx_fd(tx_fd) {}
    };

    class connection_thread_pool {

    public:
        explicit connection_thread_pool(int N);

        ~connection_thread_pool();

        void send_message(int tid, client_conn_desc *data) const;

        inline int size() const { return pool_size; }

    private:
        const int pool_size;

        std::thread **threads;
        std::queue<client_conn_desc *> **queues;
        std::mutex **mutexes;
        ingress::semaphore **signals;

        [[noreturn]]
        static void worker_code(std::queue<client_conn_desc *> &job_queue, ingress::semaphore &available,
                                std::mutex &exclusive);

    };

}

#endif //CLUSTER_MANAGER_THREAD_POOL_H

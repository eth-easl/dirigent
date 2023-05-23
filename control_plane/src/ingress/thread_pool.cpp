//
// Created by lcvetkovic on 5/16/23.
//

#include "thread_pool.h"

ingress::connection_thread_pool::connection_thread_pool(const int N) : pool_size(N) {
    threads = new std::thread *[N];
    queues = new std::queue<client_conn_desc *> *[N];
    mutexes = new std::mutex *[N];
    signals = new ingress::semaphore *[N];

    for (int i = 0; i < N; i++) {
        auto queue = new std::queue<client_conn_desc *>();
        auto mutex = new std::mutex();
        auto semaphore = new ingress::semaphore(0);

        threads[i] = new std::thread(ingress::connection_thread_pool::worker_code,
                                     std::ref(*queue), std::ref(*semaphore), std::ref(*mutex));
        queues[i] = queue;
        mutexes[i] = mutex;
        signals[i] = semaphore;
    }
}

ingress::connection_thread_pool::~connection_thread_pool() {
    for (int i = 0; i < pool_size; i++) {
        delete threads[i];
        delete queues[i];
        delete mutexes[i];
        delete signals[i];
    }

    delete[] threads;
    delete[] queues;
    delete[] mutexes;
    delete[] signals;
}

void ingress::connection_thread_pool::send_message(const int tid, client_conn_desc *const data) const {
    mutexes[tid]->lock();
    queues[tid]->push(data);
    mutexes[tid]->unlock();

    signals[tid]->release();

    spdlog::debug("Message sent to connection handler thread {}", tid);
}

void
ingress::connection_thread_pool::worker_code(std::queue<client_conn_desc *> &job_queue,
                                             ingress::semaphore &available,
                                             std::mutex &exclusive) {
    while (true) {
        available.acquire();

        exclusive.lock();
        auto conn_desc = job_queue.front();
        job_queue.pop();
        exclusive.unlock();

        spdlog::debug("Message received to connection handler thread");

        /*vector<char *> datagram;
        constexpr int BUFFER_SIZE = 4;
        char *buf = new char[BUFFER_SIZE];
        // TODO(lazar): remember to deallocate this memory somewhere

        int bytes_read;
        while ((bytes_read = read(conn_desc->tx_fd, buf, BUFFER_SIZE)) != 0) {
            datagram.push_back(buf);
            //cout << string(buf) << endl;
            buf = new char[BUFFER_SIZE];
        }*/



        close(conn_desc->tx_fd);
        delete conn_desc;
    }
}

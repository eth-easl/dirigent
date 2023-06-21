//
// Created by lcvetkovic on 5/16/23.
//

#ifndef CLUSTER_MANAGER_INGRESS_SEMAPHORE_H
#define CLUSTER_MANAGER_INGRESS_SEMAPHORE_H

#include <mutex>
#include <condition_variable>

namespace ingress {

    class semaphore {
        std::mutex mutex_;
        std::condition_variable condition_;
        unsigned long count_;

    public:
        explicit semaphore(int init_count) : count_(init_count) {}

        void release() {
            std::lock_guard<decltype(mutex_)> lock(mutex_);
            ++count_;
            condition_.notify_one();
        }

        void acquire() {
            std::unique_lock<decltype(mutex_)> lock(mutex_);
            while (!count_) // Handle spurious wake-ups.
                condition_.wait(lock);
            --count_;
        }

        [[maybe_unused]]
        bool try_acquire() {
            std::lock_guard<decltype(mutex_)> lock(mutex_);
            if (count_) {
                --count_;
                return true;
            }
            return false;
        }
    };

}

#endif //CLUSTER_MANAGER_INGRESS_SEMAPHORE_H

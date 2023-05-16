//
// Created by lcvetkovic on 5/11/23.
//

#ifndef CLUSTER_MANAGER_REQUEST_BUFFER_H
#define CLUSTER_MANAGER_REQUEST_BUFFER_H

#include <iostream>

using namespace std;

namespace ingress {
    template<typename T>
    class RequestBuffer {

    public:
        ~RequestBuffer() {
            while (head != nullptr) {
                node *n = head;
                head = head->next;
                delete n;
            }
        }

        void Enqueue(T data) {
            node *n = new node();
            n->data = data;
            n->next = nullptr;

            if (head == nullptr)
                head = tail = n;
            else {
                tail->next = n;
                tail = n;
            }

            length++;
        }

        T Dequeue() {
            if (head == nullptr)
                throw runtime_error("Dequeue called on an empty buffer.");

            T ret = head->data;
            node *next = head->next;

            delete head;
            head = next;

            if (--length == 0)
                tail = nullptr;

            return ret;
        }

        [[nodiscard]]
        size_t Length() const {
            return length;
        }

        [[nodiscard]]
        bool Empty() const {
            return head == nullptr;
        }

    private:
        struct node {
            T data;
            node *next;
        };

        node *head = nullptr;
        node *tail = nullptr;

        uint32_t length = 0;
    };
}

#endif //CLUSTER_MANAGER_REQUEST_BUFFER_H

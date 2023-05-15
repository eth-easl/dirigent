//
// Created by lcvetkovic on 5/11/23.
//

#ifndef CLUSTER_MANAGER_REQUEST_BUFFER_H
#define CLUSTER_MANAGER_REQUEST_BUFFER_H

#include <iostream>

using namespace std;

template<typename T>
class RequestBuffer {

public:
    ~RequestBuffer();

    void Enqueue(const T &);

    const T &Dequeue();

    uint32_t Length() const;

    bool Empty() const;

private:
    struct node {
        const T &data;
        node *next;
    };

    node *head = nullptr;
    node *tail = nullptr;

    uint32_t length;
};

#endif //CLUSTER_MANAGER_REQUEST_BUFFER_H

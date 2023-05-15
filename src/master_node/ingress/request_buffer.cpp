//
// Created by lcvetkovic on 5/11/23.
//

#include "request_buffer.h"

template<typename T>
RequestBuffer<T>::~RequestBuffer() {
    while (head != nullptr) {
        node *n = head;
        head = head->next;
        delete n;
    }
}

template<typename T>
void RequestBuffer<T>::Enqueue(const T &data) {
    node *n = new node();
    n->data = data;
    n->next = nullptr;

    if (head == nullptr)
        head = tail = n;
    else
        tail->next = n;

    length++;
}

template<typename T>
const T &RequestBuffer<T>::Dequeue() {
    if (head == nullptr)
        throw runtime_error("Dequeue called on an empty buffer.");

    const T &ret = head->data;
    head = head->next;

    length--;

    return ret;
}

template<typename T>
uint32_t RequestBuffer<T>::Length() const {
    return length;
}

template<typename T>
bool RequestBuffer<T>::Empty() const {
    return head == nullptr;
}
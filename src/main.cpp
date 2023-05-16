#include <iostream>

#include "ingress/endpoint.h"
#include "ingress/request_buffer.h"

using namespace ingress;
using namespace std;

//constexpr uint WORKER_METRICS_RX_PORT = 9091;

int main() {
    RequestBuffer<int> buffer;
    //Ingress ingress(buffer);

    //ingress.startServing();

    return 0;
}

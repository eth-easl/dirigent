#include <iostream>

#include "ingress/endpoint.h"

using namespace master_node::ingress;
using namespace std;

//constexpr uint WORKER_METRICS_RX_PORT = 9091;

int main() {
    master_node::ingress::serve();

    return 0;
}

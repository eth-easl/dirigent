#include <iostream>

#include "ingress/endpoint.h"

using namespace ingress;
using namespace std;

//constexpr uint WORKER_METRICS_RX_PORT = 9091;

int main() {
    spdlog::set_level(spdlog::level::debug);

    Ingress ingress_controller;
    ingress_controller.start_serving();
}

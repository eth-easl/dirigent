cmake_minimum_required(VERSION 3.25)
project(cluster_manager)

set(CMAKE_CXX_STANDARD 17)

add_executable(cluster_manager src/master_node/main.cpp src/master_node/ingress/endpoint.cpp)

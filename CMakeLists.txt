cmake_minimum_required(VERSION 3.25)
project(cluster_manager)

set(CMAKE_CXX_STANDARD 17)

# Include Google Tests
include(FetchContent)
FetchContent_Declare(
        googletest
        URL https://github.com/google/googletest/archive/refs/tags/v1.13.0.zip
)
FetchContent_MakeAvailable(googletest)
add_library(GTest::GTest INTERFACE IMPORTED)

include_directories(src)

add_subdirectory(src)
add_subdirectory(tests)

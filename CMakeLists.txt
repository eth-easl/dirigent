cmake_minimum_required(VERSION 3.25)
project(cluster_manager)

set(CMAKE_CXX_STANDARD 17)
include(FetchContent)

# Include Google Tests
FetchContent_Declare(
        googletest
        URL https://github.com/google/googletest/archive/refs/tags/v1.13.0.zip
)
FetchContent_MakeAvailable(googletest)
add_library(GTest::GTest INTERFACE IMPORTED)

# Include Google Tests
FetchContent_Declare(
        spdlog
        GIT_REPOSITORY https://github.com/gabime/spdlog.git
        GIT_TAG v1.11.0)
FetchContent_GetProperties(spdlog)
if (NOT spdlog_POPULATED)
    FetchContent_Populate(spdlog)
    add_subdirectory(${spdlog_SOURCE_DIR} ${spdlog_BINARY_DIR})
endif ()

# Including source files
include_directories(src)

add_subdirectory(src)
add_subdirectory(tests)

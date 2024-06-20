//
// Created by lcvetkovic on 6/20/24.
//

#ifndef WORKLOAD_MAIN_H
#define WORKLOAD_MAIN_H

#include <boost/algorithm/string.hpp>
#include <boost/asio.hpp>
#include <boost/asio/ip/host_name.hpp>
#include <boost/beast.hpp>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <string>

using tcp = boost::asio::ip::tcp;
namespace http = boost::beast::http;

std::string encodeJSON(const std::string &status, const std::string &function,
                       const std::string &machine_name, int execution_time);

void workload_function(long total_iterations);

void decode_header(http::request<http::string_body> &request, std::string &function,
                   std::string &workload, long &requested_cpu, long &multiplier);

void handle_request(http::request<http::string_body> &request, tcp::socket &socket);

void main_handler(http::request<http::string_body> &request, http::response<http::string_body> &response);

void health_handler(http::response<http::string_body> &response);

[[noreturn]] void runServer();

#endif //WORKLOAD_MAIN_H

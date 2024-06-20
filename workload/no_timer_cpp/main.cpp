//
// Created by lcvetkovic on 6/20/24.
//

#include "main.h"

static std::string hostname = boost::asio::ip::host_name();

std::string encodeJSON(const std::string &status, const std::string &function, const std::string &machine_name,
                       int execution_time) {
    std::stringstream os;

    os << R"({"Status":")";
    os << status;
    os << R"(","Function":")";
    os << function;
    os << R"(","MachineName":")";
    os << machine_name;
    os << R"(","ExecutionTime":)";
    os << execution_time;
    os << "}";

    return os.str();
}

void workload_function(long total_iterations) {
    volatile double result = 0.0;
    volatile double input = 10.0;
    volatile long iteration;

    for (iteration = 0; iteration < total_iterations; iteration++) {
        result = sqrt(input);
    }

    std::cout << result << std::endl;
}

void decode_header(http::request<http::string_body> &request, std::string &function,
                   std::string &workload, long &requested_cpu, long &multiplier) {
    for (auto &h: request.base()) {
        std::string param_name = std::string(h.name_string());
        boost::algorithm::to_lower(param_name);

#if DEBUG
        std::cout << param_name << " " << h.value() << std::endl;
#endif

        if (param_name == "workload") {
            workload = std::string(h.value());
        } else if (param_name == "function") {
            function = std::string(h.value());
        } else if (param_name == "requested_cpu") {
            requested_cpu = std::strtol(std::string(h.value()).c_str(), nullptr, 10);
        } else if (param_name == "multiplier") {
            multiplier = std::strtol(std::string(h.value()).c_str(), nullptr, 10);
        }
    }
}

void main_handler(http::request<http::string_body> &request, http::response<http::string_body> &response) {
    std::string workload;
    std::string function;
    long requested_cpu;
    long multiplier;

    decode_header(request, function, workload, requested_cpu, multiplier);

    if (workload == "empty") {
        std::string rawResponse = encodeJSON("OK - EMPTY", function, hostname, 0);

        response.body() = rawResponse;
        response.result(http::status::ok);
    } else if (workload == "trace") {
        workload_function(requested_cpu * multiplier);
        std::string rawResponse = encodeJSON("OK", function, hostname, 0);

        response.body() = rawResponse;
        response.result(http::status::ok);
    } else {
        std::cout << request << " "
                  << workload << " "
                  << function << " "
                  << requested_cpu << " "
                  << multiplier << std::endl;

        response.result(http::status::bad_request);
    }
}

void handle_request(http::request<http::string_body> &request, tcp::socket &socket) {
    http::response<http::string_body> response;
    response.version(request.version());

    std::string path = std::string(request.target());

    if (path == "/") {
        main_handler(request, response);
    } else if (path == "/health") {
        health_handler(response);
    } else {
        response.result(http::status::bad_request);
    }

    response.prepare_payload();
    boost::beast::http::write(socket, response);
}

void health_handler(http::response<http::string_body> &response) {
    response.result(http::status::ok);
}

[[noreturn]] void runServer() {
    boost::asio::io_context io_context;
    tcp::acceptor acceptor(io_context, {tcp::v4(), 80});

    while (true) {
        tcp::socket socket(io_context);
        acceptor.accept(socket);

        // Read the HTTP request
        boost::beast::flat_buffer buffer;
        http::request<http::string_body> request;
        boost::beast::http::read(socket, buffer, request);

        // Handle the request
        handle_request(request, socket);

        // Close the socket
        socket.shutdown(tcp::socket::shutdown_send);
    }
}

int main() {
    try {
        runServer();
    } catch (const std::exception &e) {
        std::cerr << "Exception: " << e.what() << std::endl;
    }

    return 0;
}
/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package utils

import "time"

const (
	Localhost string = "0.0.0.0"

	DefaultDataPlaneProxyPort string = "8080"
	DefaultDataPlaneApiPort   string = "8081"

	DefaultControlPlanePort                    string = "9090"
	DefaultControlPlanePortServiceRegistration string = "9091"

	DefaultWorkerNodePort int = 10010

	DefaultTraceOutputFolder string = "data"

	TCP string = "tcp"

	TestDockerImageName string = "docker.io/cvetkovic/dirigent_empty_function:latest"

	TolerateHeartbeatMisses = 3
	HeartbeatInterval       = 5 * time.Second

	WorkerNodeTrafficTimeout = 60 * time.Second

	GRPCConnectionTimeout = 5 * time.Second
	GRPCFunctionTimeout   = 15 * time.Minute // AWS Lambda
)

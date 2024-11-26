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

package haproxy

import (
	clientnative "github.com/haproxytech/client-native"
	"github.com/haproxytech/client-native/configuration"
	"github.com/haproxytech/client-native/runtime"
	"github.com/sirupsen/logrus"
)

func getConfigClient() *configuration.Client {
	confClient := &configuration.Client{}

	err := confClient.Init(configuration.ClientParams{
		ConfigurationFile:      "/etc/haproxy/haproxy.cfg",
		Haproxy:                "/usr/sbin/haproxy",
		UseValidation:          true,
		PersistentTransactions: true,
		TransactionDir:         "/tmp/haproxy",
	})
	if err != nil {
		logrus.Warn("Error setting up HAProxy configuration client; trying with the default one...")

		confClient, err = configuration.DefaultClient()
		if err != nil {
			logrus.Fatalf("Error setting up default HAProxy configuration client; exiting... : %s", err.Error())
		}
	}

	return confClient
}

func getRuntimeClient(configClient *configuration.Client) *runtime.Client {
	runtimeClient := &runtime.Client{}

	_, globalConf, err := configClient.GetGlobalConfiguration("")
	if err != nil {
		logrus.Warn("Cannot read HAProxy runtime API configuration; resorting to default...")
		return nil
	}

	if len(globalConf.RuntimeApis) != 0 {
		socketList := make([]string, 0, 1)
		for _, r := range globalConf.RuntimeApis {
			socketList = append(socketList, *r.Address)
		}

		if err = runtimeClient.Init(socketList, "", 0); err != nil {
			logrus.Warn("Error setting up HAProxy runtime client, not using one")
			return nil
		}
	} else {
		logrus.Warn("HAProxy runtime API not configured, not using it")
		return nil
	}

	return runtimeClient
}

func getHAProxyClient() *clientnative.HAProxyClient {
	client := &clientnative.HAProxyClient{}

	configClient := getConfigClient()
	runtimeClient := getRuntimeClient(configClient)

	err := client.Init(configClient, runtimeClient)
	if err != nil {
		logrus.Fatal("Failed to create HAProxy client.")
	}

	return client
}

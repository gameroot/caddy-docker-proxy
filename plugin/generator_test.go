package plugin

import (
	"bytes"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/assert"
)

var caddyNetworkID = "af9700b7abaab83e0a41692e02d3f74b5f5a13af877a223e9b87bd46232ee77c"

func init() {
	caddyNetworks = map[string]bool{}
	caddyNetworks[caddyNetworkID] = true
}

func TestAddContainerWithTemplates(t *testing.T) {
	var container = &types.Container{
		Names: []string{
			"container-name",
		},
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"other-network": &network.EndpointSettings{
					IPAddress: "10.0.0.1",
					NetworkID: "other-network-id",
				},
				"caddy-network": &network.EndpointSettings{
					IPAddress: "172.17.0.2",
					NetworkID: caddyNetworkID,
				},
			},
		},
		Labels: map[string]string{
			"caddy":       "{{index .Names 0}}.testdomain.com",
			"caddy.proxy": "/ {{(index .NetworkSettings.Networks \"caddy-network\").IPAddress}}:5000/api",
		},
	}

	const expected string = "container-name.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5000/api\n" +
		"}\n"

	testSingleContainer(t, container, expected)
}
func TestAddContainerWithBasicLabels(t *testing.T) {
	var container = &types.Container{
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"other-network": &network.EndpointSettings{
					IPAddress: "10.0.0.1",
					NetworkID: "other-network-id",
				},
				"caddy-network": &network.EndpointSettings{
					IPAddress: "172.17.0.2",
					NetworkID: caddyNetworkID,
				},
			},
		},
		Labels: map[string]string{
			"caddy.address":    "service.testdomain.com",
			"caddy.targetport": "5000",
			"caddy.targetpath": "/api",
		},
	}

	const expected string = "service.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5000/api\n" +
		"}\n"

	testSingleContainer(t, container, expected)
}

func TestAddContainerDifferentNetwork(t *testing.T) {
	var container = &types.Container{
		ID: "CONTAINER-ID",
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"other-network": &network.EndpointSettings{
					IPAddress: "10.0.0.1",
					NetworkID: "other-network-id",
				},
			},
		},
		Labels: map[string]string{
			"caddy.address":    "service.testdomain.com",
			"caddy.targetport": "5000",
			"caddy.targetpath": "/api",
		},
	}

	const expected string = "# Container CONTAINER-ID and caddy are not in same network\n"

	testSingleContainer(t, container, expected)
}

func TestAddContainerWithBasicLabelsAndMultipleConfigs(t *testing.T) {
	var container = &types.Container{
		NetworkSettings: &types.SummaryNetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"other-network": &network.EndpointSettings{
					IPAddress: "10.0.0.1",
					NetworkID: "other-network-id",
				},
				"caddy-network": &network.EndpointSettings{
					IPAddress: "172.17.0.2",
					NetworkID: caddyNetworkID,
				},
			},
		},
		Labels: map[string]string{
			"caddy_0.address":    "service1.testdomain.com",
			"caddy_0.targetport": "5000",
			"caddy_0.targetpath": "/api",
			"caddy_0.tls.dns":    "route53",
			"caddy_1.address":    "service2.testdomain.com",
			"caddy_1.targetport": "5001",
			"caddy_1.tls.dns":    "route53",
		},
	}

	const expected string = "service1.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5000/api\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n" +
		"service2.testdomain.com {\n" +
		"  proxy / 172.17.0.2:5001\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testSingleContainer(t, container, expected)
}

func TestAddServiceWithTemplates(t *testing.T) {
	var service = &swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy":                    "{{.Spec.Name}}.testdomain.com",
					"caddy.proxy":              "/ {{.Spec.Name}}:5000/api",
					"caddy.proxy.transparent":  "",
					"caddy.proxy.health_check": "/health",
					"caddy.proxy.websocket":    "",
					"caddy.gzip":               "",
					"caddy.basicauth":          "/ user password",
					"caddy.tls.dns":            "route53",
					"caddy.rewrite_0":          "/path1 /path2",
					"caddy.rewrite_1":          "/path3 /path4",
					"caddy.limits.header":      "100kb",
					"caddy.limits.body_0":      "/path1 2mb",
					"caddy.limits.body_1":      "/path2 4mb",
				},
			},
		},
		Endpoint: swarm.Endpoint{
			VirtualIPs: []swarm.EndpointVirtualIP{
				swarm.EndpointVirtualIP{
					NetworkID: caddyNetworkID,
				},
			},
		},
	}

	const expected string = "service.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  gzip\n" +
		"  limits {\n" +
		"    body /path1 2mb\n" +
		"    body /path2 4mb\n" +
		"    header 100kb\n" +
		"  }\n" +
		"  proxy / service:5000/api {\n" +
		"    health_check /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  rewrite /path1 /path2\n" +
		"  rewrite /path3 /path4\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testSingleService(t, false, service, expected)
}

func TestAddServiceWithBasicLabels(t *testing.T) {
	var service = &swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy.address":            "service.testdomain.com",
					"caddy.targetport":         "5000",
					"caddy.targetpath":         "/api",
					"caddy.proxy.health_check": "/health",
					"caddy.proxy.transparent":  "",
					"caddy.proxy.websocket":    "",
					"caddy.basicauth":          "/ user password",
					"caddy.tls.dns":            "route53",
				},
			},
		},
		Endpoint: swarm.Endpoint{
			VirtualIPs: []swarm.EndpointVirtualIP{
				swarm.EndpointVirtualIP{
					NetworkID: caddyNetworkID,
				},
			},
		},
	}

	const expected string = "service.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  proxy / service:5000/api {\n" +
		"    health_check /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testSingleService(t, false, service, expected)
}

func TestAddServiceWithBasicLabelsAndMultipleConfigs(t *testing.T) {
	var service = &swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy_0.address":            "service1.testdomain.com",
					"caddy_0.targetport":         "5000",
					"caddy_0.targetpath":         "/api",
					"caddy_0.proxy.health_check": "/health",
					"caddy_0.proxy.transparent":  "",
					"caddy_0.proxy.websocket":    "",
					"caddy_0.basicauth":          "/ user password",
					"caddy_0.tls.dns":            "route53",
					"caddy_1.address":            "service2.testdomain.com",
					"caddy_1.targetport":         "5001",
					"caddy_1.tls.dns":            "route53",
				},
			},
		},
		Endpoint: swarm.Endpoint{
			VirtualIPs: []swarm.EndpointVirtualIP{
				swarm.EndpointVirtualIP{
					NetworkID: caddyNetworkID,
				},
			},
		},
	}

	const expected string = "service1.testdomain.com {\n" +
		"  basicauth / user password\n" +
		"  proxy / service:5000/api {\n" +
		"    health_check /health\n" +
		"    transparent\n" +
		"    websocket\n" +
		"  }\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n" +
		"service2.testdomain.com {\n" +
		"  proxy / service:5001\n" +
		"  tls {\n" +
		"    dns route53\n" +
		"  }\n" +
		"}\n"

	testSingleService(t, false, service, expected)
}

func TestAddServiceProxyServiceTasks(t *testing.T) {
	var service = &swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy.address":    "service.testdomain.com",
					"caddy.targetport": "5000",
				},
			},
		},
		Endpoint: swarm.Endpoint{
			VirtualIPs: []swarm.EndpointVirtualIP{
				swarm.EndpointVirtualIP{
					NetworkID: caddyNetworkID,
				},
			},
		},
	}

	const expected string = "service.testdomain.com {\n" +
		"  proxy / tasks.service:5000\n" +
		"}\n"

	testSingleService(t, true, service, expected)
}

func TestAddServiceDifferentNetwork(t *testing.T) {
	var service = &swarm.Service{
		ID: "SERVICE-ID",
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy.address":    "service.testdomain.com",
					"caddy.targetport": "5000",
				},
			},
		},
		Endpoint: swarm.Endpoint{
			VirtualIPs: []swarm.EndpointVirtualIP{
				swarm.EndpointVirtualIP{
					NetworkID: "other-network-id",
				},
			},
		},
	}

	const expected string = "# Service SERVICE-ID and caddy are not in same network\n"

	testSingleService(t, false, service, expected)
}

func TestIgnoreLabelsWithoutCaddyPrefix(t *testing.T) {
	var service = &swarm.Service{
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "service",
				Labels: map[string]string{
					"caddy_version":  "0.11.0",
					"caddyversion":   "0.11.0",
					"caddy_.version": "0.11.0",
					"version_caddy":  "0.11.0",
				},
			},
		},
		Endpoint: swarm.Endpoint{
			VirtualIPs: []swarm.EndpointVirtualIP{
				swarm.EndpointVirtualIP{
					NetworkID: caddyNetworkID,
				},
			},
		},
	}

	const expected string = ""

	testSingleService(t, true, service, expected)
}

func testSingleService(t *testing.T, shouldProxyServiceTasks bool, service *swarm.Service, expected string) {
	var buffer bytes.Buffer
	proxyServiceTasks = shouldProxyServiceTasks
	addServiceToCaddyFile(&buffer, service)
	var content = buffer.String()
	assert.Equal(t, expected, content)
}

func testSingleContainer(t *testing.T, container *types.Container, expected string) {
	var buffer bytes.Buffer
	addContainerToCaddyFile(&buffer, container)
	var content = buffer.String()
	assert.Equal(t, expected, content)
}

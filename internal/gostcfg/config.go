package gostcfg

import "strconv"

type Config struct {
	Services  []Service  `json:"services"`
	Chains    []Chain    `json:"chains,omitempty"`
	Resolvers []Resolver `json:"resolvers,omitempty"`
}

type Service struct {
	Name      string   `json:"name"`
	Addr      string   `json:"addr"`
	Handler   Handler  `json:"handler"`
	Listener  Listener `json:"listener"`
	Interface string   `json:"interface,omitempty"`
	Resolver  string   `json:"resolver,omitempty"`
}

type Handler struct {
	Type    string `json:"type"`
	Auth    *Auth  `json:"auth,omitempty"`
	Chain   string `json:"chain,omitempty"`
	Retries int    `json:"retries,omitempty"`
}

type Listener struct {
	Type string `json:"type"`
	Auth *Auth  `json:"auth,omitempty"`
}

type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Chain struct {
	Name     string   `json:"name"`
	Selector Selector `json:"selector,omitempty"`
	Hops     []Hop    `json:"hops"`
}

type Hop struct {
	Name     string   `json:"name"`
	Selector Selector `json:"selector,omitempty"`
	Nodes    []Node   `json:"nodes"`
}

type Node struct {
	Name      string    `json:"name"`
	Addr      string    `json:"addr"`
	Connector Connector `json:"connector"`
	Dialer    Dialer    `json:"dialer"`
}

type Connector struct {
	Type string `json:"type"`
	Auth *Auth  `json:"auth,omitempty"`
}

type Dialer struct {
	Type string `json:"type"`
}

type Selector struct {
	Strategy    string `json:"strategy,omitempty"`
	MaxFails    int    `json:"maxFails,omitempty"`
	FailTimeout string `json:"failTimeout,omitempty"`
}

type Resolver struct {
	Name        string       `json:"name"`
	Nameservers []Nameserver `json:"nameservers"`
}

type Nameserver struct {
	Addr   string `json:"addr"`
	Only   string `json:"only,omitempty"`
	Prefer string `json:"prefer,omitempty"`
}

func NodeProxy(httpPort, socksPort int, username, password, egressInterface, resolverOnly string) Config {
	auth := &Auth{Username: username, Password: password}
	if resolverOnly == "prefer_ipv6" {
		return nodeProxyPreferIPv6Fallback(httpPort, socksPort, auth)
	}
	resolverName, resolvers := nodeResolvers(resolverOnly)
	services := make([]Service, 0, 2)
	if httpPort > 0 {
		services = append(services, Service{
			Name: "node-http",
			Addr: ":" + strconv.Itoa(httpPort),
			Handler: Handler{
				Type: "http",
				Auth: auth,
			},
			Listener:  Listener{Type: "tcp"},
			Interface: egressInterface,
			Resolver:  resolverName,
		})
	}
	if socksPort > 0 {
		services = append(services, Service{
			Name: "node-socks5",
			Addr: ":" + strconv.Itoa(socksPort),
			Handler: Handler{
				Type: "socks5",
				Auth: auth,
			},
			Listener:  Listener{Type: "tcp"},
			Interface: egressInterface,
			Resolver:  resolverName,
		})
	}
	return Config{Services: services, Resolvers: resolvers}
}

func nodeProxyPreferIPv6Fallback(httpPort, socksPort int, auth *Auth) Config {
	const (
		chainName     = "node-prefer-ipv6-fallback"
		resolverIPv6  = "resolver-ipv6"
		resolverIPv4  = "resolver-ipv4"
		localIPv6Name = "node-prefer-ipv6-egress-v6"
		localIPv4Name = "node-prefer-ipv6-egress-v4"
	)
	v6Port, v4Port := fallbackLoopbackPorts(httpPort, socksPort)
	v6Addr := "127.0.0.1:" + strconv.Itoa(v6Port)
	v4Addr := "127.0.0.1:" + strconv.Itoa(v4Port)

	services := make([]Service, 0, 4)
	if httpPort > 0 {
		services = append(services, Service{
			Name: "node-http",
			Addr: ":" + strconv.Itoa(httpPort),
			Handler: Handler{
				Type:    "http",
				Auth:    auth,
				Chain:   chainName,
				Retries: 1,
			},
			Listener: Listener{Type: "tcp"},
		})
	}
	if socksPort > 0 {
		services = append(services, Service{
			Name: "node-socks5",
			Addr: ":" + strconv.Itoa(socksPort),
			Handler: Handler{
				Type:    "socks5",
				Auth:    auth,
				Chain:   chainName,
				Retries: 1,
			},
			Listener: Listener{Type: "tcp"},
		})
	}
	if len(services) == 0 {
		return Config{}
	}
	services = append(services,
		Service{
			Name: localIPv6Name,
			Addr: v6Addr,
			Handler: Handler{
				Type: "socks5",
				Auth: auth,
			},
			Listener: Listener{Type: "tcp"},
			Resolver: resolverIPv6,
		},
		Service{
			Name: localIPv4Name,
			Addr: v4Addr,
			Handler: Handler{
				Type: "socks5",
				Auth: auth,
			},
			Listener: Listener{Type: "tcp"},
			Resolver: resolverIPv4,
		},
	)

	selector := Selector{
		Strategy:    "fifo",
		MaxFails:    1,
		FailTimeout: "30s",
	}
	return Config{
		Services: services,
		Chains: []Chain{{
			Name:     chainName,
			Selector: selector,
			Hops: []Hop{{
				Name:     "node-prefer-ipv6-hop",
				Selector: selector,
				Nodes: []Node{
					{
						Name: localIPv6Name,
						Addr: v6Addr,
						Connector: Connector{
							Type: "socks5",
							Auth: auth,
						},
						Dialer: Dialer{Type: "tcp"},
					},
					{
						Name: localIPv4Name,
						Addr: v4Addr,
						Connector: Connector{
							Type: "socks5",
							Auth: auth,
						},
						Dialer: Dialer{Type: "tcp"},
					},
				},
			}},
		}},
		Resolvers: []Resolver{
			resolver("ipv6"),
			resolver("ipv4"),
		},
	}
}

func fallbackLoopbackPorts(httpPort, socksPort int) (int, int) {
	used := map[int]bool{}
	if httpPort > 0 {
		used[httpPort] = true
	}
	if socksPort > 0 {
		used[socksPort] = true
	}
	candidates := []int{
		61080, 61081,
		62080, 62081,
		60080, 60081,
		59080, 59081,
		58080, 58081,
	}
	for _, seed := range []int{httpPort, socksPort, 18080} {
		if seed <= 0 {
			continue
		}
		for _, delta := range []int{10000, 20000, 30000, 40000} {
			for _, port := range []int{seed + delta, seed + delta + 1} {
				if port > 0 && port <= 65535 {
					candidates = append(candidates, port)
				}
			}
		}
	}

	var ports []int
	for _, port := range candidates {
		if port <= 0 || port > 65535 || used[port] {
			continue
		}
		used[port] = true
		ports = append(ports, port)
		if len(ports) == 2 {
			return ports[0], ports[1]
		}
	}
	return 61080, 61081
}

func nodeResolvers(only string) (string, []Resolver) {
	if only == "prefer_ipv6" {
		name := "resolver-prefer-ipv6"
		return name, []Resolver{{
			Name: name,
			Nameservers: []Nameserver{{
				Addr:   "1.1.1.1",
				Prefer: "ipv6",
			}},
		}}
	}
	if only != "ipv4" && only != "ipv6" {
		return "", nil
	}
	name := "resolver-" + only
	return name, []Resolver{resolver(only)}
}

func resolver(only string) Resolver {
	return Resolver{
		Name: "resolver-" + only,
		Nameservers: []Nameserver{{
			Addr: "1.1.1.1",
			Only: only,
		}},
	}
}

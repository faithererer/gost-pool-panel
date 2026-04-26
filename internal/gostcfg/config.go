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
	Type  string `json:"type"`
	Auth  *Auth  `json:"auth,omitempty"`
	Chain string `json:"chain,omitempty"`
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
	return name, []Resolver{{
		Name: name,
		Nameservers: []Nameserver{{
			Addr: "1.1.1.1",
			Only: only,
		}},
	}}
}

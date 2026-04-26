package gostcfg

import "strconv"

type Config struct {
	Services []Service `json:"services"`
	Chains   []Chain   `json:"chains,omitempty"`
}

type Service struct {
	Name     string         `json:"name"`
	Addr     string         `json:"addr"`
	Handler  Handler        `json:"handler"`
	Listener Listener       `json:"listener"`
	Metadata map[string]any `json:"metadata,omitempty"`
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

func NodeProxy(httpPort, socksPort int, username, password, egressInterface string) Config {
	auth := &Auth{Username: username, Password: password}
	metadata := serviceMetadata(egressInterface)
	services := make([]Service, 0, 2)
	if httpPort > 0 {
		services = append(services, Service{
			Name: "node-http",
			Addr: ":" + strconv.Itoa(httpPort),
			Handler: Handler{
				Type: "http",
				Auth: auth,
			},
			Listener: Listener{Type: "tcp"},
			Metadata: metadata,
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
			Listener: Listener{Type: "tcp"},
			Metadata: metadata,
		})
	}
	return Config{Services: services}
}

func serviceMetadata(egressInterface string) map[string]any {
	if egressInterface == "" {
		return nil
	}
	return map[string]any{"interface": egressInterface}
}

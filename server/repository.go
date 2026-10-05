package server

import (
	"sync"

	"github.com/evan-buss/openbooks/core"
)

type Repository struct {
	mutex   sync.RWMutex
	servers core.IrcServers
}

func NewRepository() *Repository {
	return &Repository{servers: core.IrcServers{}}
}

func (r *Repository) Servers() core.IrcServers {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.servers
}

func (r *Repository) SetServers(servers core.IrcServers) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.servers = servers
}

package backends

type Backend struct {
	Ip   string
	Reqs int32 // active requests to this backend currently
}

func NewBackend(ip string) *Backend {
	return &Backend{Ip: ip}
}

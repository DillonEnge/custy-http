package custyhttp

type Router struct {
	server *Server
}

func NewRouter(s *Server) *Router {
	return &Router{
		server: s,
	}
}

func (r *Router) Get(route string, f HandlerFunc) {
	r.server.registerRoute(METHOD_GET, route, f)
}

func (r *Router) Post(route string, f HandlerFunc) {
	r.server.registerRoute(METHOD_POST, route, f)
}

func (r *Router) Patch(route string, f HandlerFunc) {
	r.server.registerRoute(METHOD_PATCH, route, f)
}

func (r *Router) Put(route string, f HandlerFunc) {
	r.server.registerRoute(METHOD_PUT, route, f)
}

func (r *Router) Delete(route string, f HandlerFunc) {
	r.server.registerRoute(METHOD_DELETE, route, f)
}

func (r *Router) Options(route string, f HandlerFunc) {
	r.server.registerRoute(METHOD_OPTIONS, route, f)
}

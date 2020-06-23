package server

import (
	"net/http"
	"sync"
)

// AddRoute adds a new single fire route to the server.
func (s *Server) AddRoute(route *Route) {
	if s.wg == nil {
		s.wg = &sync.WaitGroup{}
		s.wg.Add(1)
		go func() {
			s.wg.Wait()
			s.Done <- s.finishedRoutes
			close(s.Done)
		}()
	}

	okMetric := true
	if route.MaxRequests != 0 {
		okMetric = false
	} else if route.MaxOK == 0 {
		route.MaxOK = 1
	}

	rr := s.router.HandleFunc(route.Pattern, func(w http.ResponseWriter, r *http.Request) {
		var rc int64
		var err error
		route.Lock()
		route.reqCount++

		if okMetric {
			switch {
			case route.okCount >= route.MaxOK:
				route.DoneHandlerFunc(w, r)
			case route.okCount < route.MaxOK:
				err = route.HandlerFunc(w, r)

				if err == nil || err == OKDoneErr {
					route.okCount++
					err = OKDoneErr
				} else if err != OKNotDoneErr {
					s.internalError(err.Error())
				}

				if route.okCount == route.MaxOK {
					s.Lock()
					s.finishedRoutes[route] = err
					s.Unlock()
					s.wg.Done()
				}
			}
			route.Unlock()
			return
		}

		rc = route.reqCount
		route.Unlock()
		switch {
		case rc > route.MaxRequests:
			route.DoneHandlerFunc(w, r)
		case rc <= route.MaxRequests:
			err = route.HandlerFunc(w, r)
			if err == nil || err == OKDoneErr {
				route.Lock()
				route.okCount++
				route.Unlock()
				err = OKDoneErr
			} else if err != OKNotDoneErr {
				s.internalError(err.Error())
			}

			if rc == route.MaxRequests {
				s.Lock()
				s.finishedRoutes[route] = err
				s.Unlock()
				s.wg.Done()
			}
		}
	})

	if len(route.Methods) > 0 {
		rr.Methods(route.Methods...)
	}
}

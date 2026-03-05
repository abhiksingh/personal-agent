package transport

import (
	"time"

	"github.com/gorilla/websocket"
)

type realtimeSessionReservationFailure struct {
	limitType string
	limit     int
	active    int
}

func (s *Server) reserveRealtimeSession() (bool, realtimeSessionReservationFailure) {
	if s == nil {
		return true, realtimeSessionReservationFailure{}
	}
	s.realtimeMu.Lock()
	defer s.realtimeMu.Unlock()

	if s.realtimeConnections >= s.config.RealtimeMaxConnections {
		return false, realtimeSessionReservationFailure{
			limitType: "connections",
			limit:     s.config.RealtimeMaxConnections,
			active:    s.realtimeConnections,
		}
	}
	if s.realtimeSubscriptions >= s.config.RealtimeMaxSubscriptions {
		return false, realtimeSessionReservationFailure{
			limitType: "subscriptions",
			limit:     s.config.RealtimeMaxSubscriptions,
			active:    s.realtimeSubscriptions,
		}
	}

	s.realtimeConnections++
	s.realtimeSubscriptions++
	return true, realtimeSessionReservationFailure{}
}

func (s *Server) releaseRealtimeSession() {
	if s == nil {
		return
	}
	s.realtimeMu.Lock()
	defer s.realtimeMu.Unlock()
	if s.realtimeConnections > 0 {
		s.realtimeConnections--
	}
	if s.realtimeSubscriptions > 0 {
		s.realtimeSubscriptions--
	}
}

func (s *Server) realtimeWriteJSON(conn *websocket.Conn, payload any) error {
	if conn == nil {
		return nil
	}
	if err := conn.SetWriteDeadline(time.Now().Add(s.config.RealtimeWriteTimeout)); err != nil {
		return err
	}
	return conn.WriteJSON(payload)
}

func (s *Server) realtimeWriteControl(conn *websocket.Conn, messageType int, data []byte) error {
	if conn == nil {
		return nil
	}
	if err := conn.SetWriteDeadline(time.Now().Add(s.config.RealtimeWriteTimeout)); err != nil {
		return err
	}
	return conn.WriteControl(messageType, data, time.Now().Add(s.config.RealtimeWriteTimeout))
}

func (s *Server) realtimeSessionCounts() (int, int) {
	if s == nil {
		return 0, 0
	}
	s.realtimeMu.Lock()
	defer s.realtimeMu.Unlock()
	return s.realtimeConnections, s.realtimeSubscriptions
}

package tlserver

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

func (s *server) configureRouters() {
	s.router.HandleFunc("/ping", s.handlePing()).Methods("GET")
	s.router.HandleFunc("/set", s.handleSetEvents()).Methods("POST")

	// -- TODO - разобраться с корректным завершением
	s.router.HandleFunc("/wait", s.handleWait()).Methods("GET")
}

func (s *server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, 200, map[string]string{"result": "pong"})
	}
}

func (s *server) handleSetEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			s.error(w, r, err)
			return
		}
		r.Body.Close()

		userAgent := r.UserAgent()

		unzip := !(userAgent == "1C+Enterprise/8.3")
		var e []*myfsm.BulkEvent

		if unzip {
			reader, err := gzip.NewReader(bytes.NewReader(body))
			if err != nil {
				s.error(w, r, err)
				return
			}
			defer reader.Close()

			if err := json.NewDecoder(reader).Decode(&e); err != nil {
				s.badRequest(w, r, err)
				return
			}
		} else {
			if err := json.NewDecoder(bytes.NewReader(body)).Decode(&e); err != nil {
				s.badRequest(w, r, err)
				return
			}
		}
		events := make([]myfsm.Event, len(e))
		for i, v := range e {
			events[i] = v
		}

		log.Printf("[%s (%s)]. Получено событий: %d\n", r.RemoteAddr, userAgent, len(events))

		if err := s.storage.Save(r.Context(), events); err != nil {
			s.error(w, r, err)
			log.Printf("%s: Ошибка сохранения: %s", r.RemoteAddr, err.Error())
			return
		}

		s.respond(w, r, http.StatusAccepted, map[string]string{"status": "accepted"})

	}
}

func (s *server) handleWait() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		done := make(chan struct{})

		go func() {
			timer := time.NewTimer(10 * time.Second)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				log.Println("Прервано!")
				time.Sleep(3 * time.Second)
				log.Println("УХОДИМ!")
				s.respond(w, r, http.StatusResetContent, map[string]string{"result": "прервано через контекст"})
			case <-timer.C:
				s.respond(w, r, http.StatusOK, map[string]string{"result": "обработка завершена"})
			}
			close(done)
		}()
		<-done
	}
}

func (s *server) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	s.respond(w, r, http.StatusBadRequest, map[string]string{"status": "bad request", "error": err.Error()})
}

func (s *server) error(w http.ResponseWriter, r *http.Request, err error) {
	s.respond(w, r, http.StatusInternalServerError, map[string]string{"status": "error", "error": err.Error()})
}

func (s *server) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

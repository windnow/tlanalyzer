package tlserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/windnow/tlanalyzer/internal/config"
)

type server struct {
	router *mux.Router
	ctx    context.Context
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func Start(conf *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())

	handler := &server{router: mux.NewRouter(), ctx: ctx}
	handler.configureRouters()

	s := http.Server{
		Addr:    conf.BindAddr,
		Handler: handler,
	}
	go breakListener(cancel, &s)

	s.BaseContext = func(_ net.Listener) context.Context { return ctx }

	return s.ListenAndServe()

}

func breakListener(cancel context.CancelFunc, server *http.Server) {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	fmt.Println("Получен сигнал:", sig)
	cancel() // Отменяем контекст
	if err := server.Shutdown(context.Background()); err != nil {
		log.Println("==>", err)
	}
}

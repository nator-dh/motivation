package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "./quotes.yaml", "path to quotes YAML")
	addr := flag.String("addr", "127.0.0.1:8765", "HTTP listen address")
	flag.Parse()

	notifier := &Notifier{}
	sched := NewScheduler(*configPath, notifier)
	if err := sched.Start(); err != nil {
		log.Fatalf("scheduler start: %v", err)
	}

	srv := NewServer(sched, notifier)
	httpSrv := &http.Server{Addr: *addr, Handler: srv.Routes()}

	go func() {
		log.Printf("listening on http://%s", *addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
	sched.Stop()
}

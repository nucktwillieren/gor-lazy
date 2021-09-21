package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	ws "github.com/nucktwillieren/gorlazy"
)

func init() {
}

func newEchoTransLayer() ws.TransportationLayer {
	return func(ctx *ws.Context) (*ws.Context, error) {
		ctx.AddSingleTargetPayload(ctx.Group, ctx.ID, ctx.Message)
		return ctx, nil
	}
}

func main() {
	hub := ws.NewHub("test")
	r := mux.NewRouter()

	r.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {
		ws.CreateConnection(rw, r, hub, newEchoTransLayer())
	})

	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			fmt.Println("ROUTE:", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			fmt.Println("Path regexp:", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			fmt.Println("Queries templates:", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			fmt.Println("Queries regexps:", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			fmt.Println("Methods:", strings.Join(methods, ","))
		}
		fmt.Println()
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    ":3000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
	//defer grpcConn.Close()
}

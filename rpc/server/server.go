package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	rpcjson "github.com/gorilla/rpc/v2/json2"

	"github.com/anyswap/CrossChain-Router/log"
	"github.com/anyswap/CrossChain-Router/params"
	"github.com/anyswap/CrossChain-Router/rpc/restapi"
	"github.com/anyswap/CrossChain-Router/rpc/rpcapi"
)

// StartAPIServer start api server
func StartAPIServer() {
	router := mux.NewRouter()
	initRouterSwapRouter(router)

	apiServer := params.GetRouterConfig().Server.APIServer
	apiPort := apiServer.Port
	allowedOrigins := apiServer.AllowedOrigins

	corsOptions := []handlers.CORSOption{
		handlers.AllowedMethods([]string{"GET", "POST"}),
	}
	if len(allowedOrigins) != 0 {
		corsOptions = append(corsOptions,
			handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"}),
			handlers.AllowedOrigins(allowedOrigins),
		)
	}

	log.Info("JSON RPC service listen and serving", "port", apiPort, "allowedOrigins", allowedOrigins)
	svr := http.Server{
		Addr:         fmt.Sprintf(":%v", apiPort),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      handlers.CORS(corsOptions...)(router),
	}
	go func() {
		if err := svr.ListenAndServe(); err != nil {
			log.Error("ListenAndServe error", "err", err)
		}
	}()
}

func initRouterSwapRouter(r *mux.Router) {
	rpcserver := rpc.NewServer()
	rpcserver.RegisterCodec(rpcjson.NewCodec(), "application/json")
	_ = rpcserver.RegisterService(new(rpcapi.RouterSwapAPI), "swap")

	r.Handle("/rpc", rpcserver)

	registerHandleFunc(r, "/versioninfo", restapi.VersionInfoHandler, "GET")
	registerHandleFunc(r, "/identifier", restapi.IdentifierHandler, "GET")
	registerHandleFunc(r, "/swap/register/{chainid}/{txid}", restapi.RegisterRouterSwapHandler, "POST")
	registerHandleFunc(r, "/swap/status/{chainid}/{txid}/", restapi.GetRouterSwapHandler, "GET")
	registerHandleFunc(r, "/swap/history/{chainid}/{address}", restapi.GetRouterSwapHistoryHandler, "GET")
}

type handleFuncType = func(w http.ResponseWriter, r *http.Request)

func registerHandleFunc(r *mux.Router, path string, handler handleFuncType, methods ...string) {
	for i := 0; i < len(methods); i++ {
		methods[i] = strings.ToUpper(methods[i])
	}
	isAcceptMethod := func(method string) bool {
		for _, acceptMethod := range methods {
			if method == acceptMethod {
				return true
			}
		}
		return false
	}
	allMethods := []string{"GET", "POST", "HEAD", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}
	excludedMethods := make([]string, 0, len(allMethods))
	for _, method := range allMethods {
		if !isAcceptMethod(method) {
			excludedMethods = append(excludedMethods, method)
		}
	}
	if len(methods) > 0 {
		acceptMethods := strings.Join(methods, ",")
		r.HandleFunc(path, handler).Methods(acceptMethods)
	}
	if len(excludedMethods) > 0 {
		forbidMethods := strings.Join(excludedMethods, ",")
		r.HandleFunc(path, warnHandler).Methods(forbidMethods)
	}
}

func warnHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Forbid '%v' on '%v'\n", r.Method, r.RequestURI)
}

package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/backstage/backstage/account"
	"github.com/backstage/backstage/auth"
	"github.com/backstage/backstage/errors"
	. "github.com/backstage/backstage/log"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/tylerb/graceful"
)

const (
	DEFAULT_PORT    = ":8000"
	DEFAULT_TIMEOUT = 10 * time.Second
)

type Api struct {
	auth   auth.Authenticatable
	router *mux.Router
}

func NewApi(store account.Storable) *Api {
	api := &Api{router: mux.NewRouter(), auth: auth.NewAuth(store)}
	api.Storage(store)

	api.router.HandleFunc("/", homeHandler)
	api.router.NotFoundHandler = http.HandlerFunc(api.notFoundHandler)

	//  Auth (login, logout, signup)
	auth := api.router.PathPrefix("/auth").Subrouter()
	auth.Methods("POST").Path("/login").HandlerFunc(api.userLogin)
	auth.Methods("DELETE").Path("/logout").HandlerFunc(api.userLogout)
	auth.Methods("POST").Path("/signup").HandlerFunc(api.userSignup)
	auth.Methods("PUT").Path("/password").HandlerFunc(api.userChangePassword)

	//  Private Routes
	private := mux.NewRouter()
	private.NotFoundHandler = http.HandlerFunc(api.notFoundHandler)

	// Middlewares
	api.router.PathPrefix("/api").Handler(negroni.New(
		negroni.NewRecovery(),
		negroni.HandlerFunc(api.errorMiddleware),
		negroni.HandlerFunc(api.requestIdMiddleware),
		negroni.HandlerFunc(api.authorizationMiddleware),
		negroni.HandlerFunc(api.contextClearerMiddleware),
		negroni.Wrap(private),
	))
	pr := private.PathPrefix("/api").Subrouter()

	// Users
	pr.Methods("DELETE").Path("/users").HandlerFunc(api.userDelete)

	// Teams
	teams := pr.Path("/teams").Subrouter()
	teams.Methods("POST").HandlerFunc(teamCreate)
	teams.Methods("GET").HandlerFunc(teamList)

	team := pr.PathPrefix("/teams/{alias}").Subrouter()
	team.Methods("DELETE").Path("/users").HandlerFunc(teamRemoveUsers)
	team.Methods("PUT").Path("/users").HandlerFunc(teamAddUsers)
	team.Methods("PUT").HandlerFunc(teamUpdate)
	team.Methods("DELETE").HandlerFunc(teamDelete)
	team.Methods("GET").HandlerFunc(teamInfo)

	// Services
	services := pr.Path("/services").Subrouter()
	services.Methods("POST").HandlerFunc(serviceCreate)
	services.Methods("GET").HandlerFunc(serviceList)

	service := pr.PathPrefix("/services/{subdomain}").Subrouter()
	service.Methods("GET").HandlerFunc(serviceInfo)
	service.Methods("DELETE").HandlerFunc(serviceDelete)
	service.Methods("PUT").HandlerFunc(serviceUpdate)

	return api
}

// Split Authenticate and CreateUserToken because we can override only the authentication method and still use the token method.
func (api *Api) Login(email, password string) (*account.TokenInfo, error) {
	user, ok := api.auth.Authenticate(email, password)
	if ok {
		token, err := api.auth.CreateUserToken(user)
		if err != nil {
			Logger.Warn(err.Error())
			return nil, err
		}
		return token, nil
	}

	return nil, errors.ErrAuthenticationFailed
}

func (api *Api) Handler() http.Handler {
	return api.router
}

// Allow to override the default authentication method.
// To be compatible, it is needed to implement the Authenticatable interface.
func (api *Api) SetAuth(auth auth.Authenticatable) {
	api.auth = auth
}

// Allow to override the default storage engine.
// To be compatible, it is needed to implement the Storable interface.
func (api *Api) Storage(store account.Storable) {
	account.Storage(store)
	api.auth = auth.NewAuth(store)
}

func (api *Api) Run() {
	graceful.Run(DEFAULT_PORT, DEFAULT_TIMEOUT, api.Handler())
}

func homeHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(rw, "Hello Backstage!")
}

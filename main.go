package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"

	"github.com/XeroAPI/xerogolang"
	"github.com/XeroAPI/xerogolang/accounting"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
)

var (
	provider = xerogolang.New(os.Getenv("XERO_KEY"), os.Getenv("XERO_SECRET"), "http://localhost:3000/auth/callback?provider=xero")
	store    = sessions.NewFilesystemStore(os.TempDir(), []byte("xero-example"))
)

func init() {
	goth.UseProviders(provider)
	store.MaxLength(math.MaxInt64)
	gothic.Store = store
}

var indexTemplate = `<p><a href="/auth/?provider=xero">Connect To Xero</a></p>`
var connectedTemplate = `
  <p><a href="/disconnect?provider=xero">Disconnect</a></p>
  <p>Connected to: {{.Name}}</p>
  <p><a href="/findcontacts?provider=xero">Find me some contacts please</a></p>`
var contactsTemplate = `
  <p><a href="/disconnect?provider=xero">Disconnect</a></p>
  {{range $index,$element:= .}}
  <p>ID: {{.ContactID}}</p>
  <p>Contact Number: {{.ContactNumber}}</p>
  <p>Name: {{.Name}}</p>
  <p>Status: {{.ContactStatus}}</p>
  <p>First Name: {{.FirstName}}</p>
  <p>Last Name: {{.LastName}}</p>
  <p>Email Address: {{.EmailAddress}}</p>
  <p>UpdatedDate: {{.UpdatedDateUTC}}</p>
  <p>-----------------------------------------------------</p>
  {{end}}`

func indexHandler(res http.ResponseWriter, req *http.Request) {
	t, _ := template.New("foo").Parse(indexTemplate)
	t.Execute(res, nil)
}
func authHandler(res http.ResponseWriter, req *http.Request) {
	// try to get the user without re-authenticating
	if gothUser, err := gothic.CompleteUserAuth(res, req); err == nil {
		t, _ := template.New("foo").Parse(connectedTemplate)
		t.Execute(res, gothUser)
	} else {
		gothic.BeginAuthHandler(res, req)
	}
}
func callbackHandler(res http.ResponseWriter, req *http.Request) {
	user, err := gothic.CompleteUserAuth(res, req)
	if err != nil {
		fmt.Fprintln(res, err)
		return
	}
	t, _ := template.New("foo").Parse(connectedTemplate)
	t.Execute(res, user)
}
func disconnectHandler(res http.ResponseWriter, req *http.Request) {
	gothic.Logout(res, req)
	res.Header().Set("Location", "/")
	res.WriteHeader(http.StatusTemporaryRedirect)
}
func findContactsHandler(res http.ResponseWriter, req *http.Request) {
	session, err := provider.GetSessionFromStore(req, res)
	if err != nil {
		fmt.Fprintln(res, err)
		return
	}
	c, err := accounting.FindContacts(provider, session, nil)
	if err != nil {
		fmt.Fprintln(res, err)
		return
	}
	t, _ := template.New("foo").Parse(contactsTemplate)
	t.Execute(res, c.Contacts)
}
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler).Methods("GET")
	a := r.PathPrefix("/auth").Subrouter()
	// "/auth/"
	a.HandleFunc("/", authHandler).Methods("GET")
	// "/auth/callback"
	a.HandleFunc("/callback", callbackHandler).Methods("GET")
	//"/disconnect"
	r.HandleFunc("/disconnect", disconnectHandler).Methods("GET")
	//"/findcontacts"
	r.HandleFunc("/findcontacts", findContactsHandler).Methods("GET")
	http.Handle("/", r)
	http.ListenAndServe(":3000", nil)
}

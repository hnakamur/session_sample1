package hello

import (
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"

	"github.com/gorilla/securecookie"
	"github.com/hnakamur/gaesessions"
)

var store = gaesessions.NewMemcacheDatastoreStore("", "",
	gaesessions.DefaultNonPersistentSessionDuration,
	securecookie.GenerateRandomKey(128))

type Greeting struct {
	Author  string
	Content string
	Date    time.Time
}

func init() {
	http.HandleFunc("/", root)
	http.HandleFunc("/sign", sign)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/session2", session2Handler)
	http.HandleFunc("/tasks/removeExpiredSessions", removeExpiredSessionsHandler)
}

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	// Get a session. We're ignoring the error resulted from decoding an
	// existing session: Get() always returns a session, even if empty.
	session, _ := store.Get(r, "session-name")
	//session.Options.MaxAge = 30
	// Set some session values.
	session.Values["foo"] = "bar"
	session.Values[42] = 43
	// Save it.
	err := session.Save(r, w)
	if err != nil {
		c := appengine.NewContext(r)
		c.Errorf("session.Save failed. err=%s", err.Error())
	}
}

func session2Handler(w http.ResponseWriter, r *http.Request) {
	// Get a session. We're ignoring the error resulted from decoding an
	// existing session: Get() always returns a session, even if empty.
	session, _ := store.Get(r, "session-name")
	c := appengine.NewContext(r)
	c.Debugf("session values. foo=%s", session.Values["foo"])
	c.Debugf("session values. 42=%d", session.Values[42])
	session.Options.MaxAge = 0
	if session.Values[42] != nil {
		session.Values[42] = session.Values[42].(int) + 1
	}
	// Save it.
	err := session.Save(r, w)
	if err != nil {
		c.Errorf("session.Save failed. err=%s", err.Error())
	}
}

func removeExpiredSessionsHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	gaesessions.RemoveExpiredDatastoreSessions(c, "")
}

func guestbookKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Guestbook", "default_guestbook", 0, nil)
}

func root(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("Greeting").Ancestor(guestbookKey(c)).Order("-Date").Limit(10)
	greetings := make([]Greeting, 0, 10)
	if _, err := q.GetAll(c, &greetings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := guestbookTemplate.Execute(w, greetings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var guestbookTemplate = template.Must(template.New("book").Parse(guestbookTemplateHTML))

const guestbookTemplateHTML = `
<html>
  <body>
  {{range .}}
    {{with .Author}}
	<p><b>{{.}}</b> wrote:</p>
	{{else}}
	<p>An anonymous person wrote:</p>
	{{end}}
	<pre>{{.Content}}</pre>
  {{end}}
    <form action="/sign" method="post">
	  <div><textarea name="content" rows="3" cols="60"></textarea></div>
	  <div><input type="submit" value="Sign Guestbook"></div>
	</form>
  </body>
</html>
`

func sign(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := Greeting{
		Content: r.FormValue("content"),
		Date:    time.Now(),
	}
	if u := user.Current(c); u != nil {
		g.Author = u.String()
	}
	key := datastore.NewIncompleteKey(c, "Greeting", guestbookKey(c))
	_, err := datastore.Put(c, key, &g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

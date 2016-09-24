package myauthboss

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gopkg.in/authboss.v0"
	_ "gopkg.in/authboss.v0/auth"
	_ "gopkg.in/authboss.v0/confirm"
	_ "gopkg.in/authboss.v0/lock"
	aboauth "gopkg.in/authboss.v0/oauth2"
	_ "gopkg.in/authboss.v0/recover"
	_ "gopkg.in/authboss.v0/register"
	_ "gopkg.in/authboss.v0/remember"

	"github.com/aarondl/tpl"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/justinas/nosurf"
)

var funcs = template.FuncMap{
	"formatDate": func(date time.Time) string {
		return date.Format("2006/01/02 03:04pm")
	},
	"yield": func() string { return "" },
}

var (
	ab        = authboss.New()
	database  = NewMemStorer()
	templates = tpl.Must(tpl.Load("myauthboss/views", "myauthboss/views/partials", "layout.html.tpl", funcs))
)

func GetAuthboss() *authboss.Authboss {
	return ab
}

func SetupStorer() {
	cookieStoreKey, _ := base64.StdEncoding.DecodeString(`NpEPi8pEjKVjLGJ6kYCS+VTCzi6BUuDzU0wrwXyf5uDPArtlofn2AG6aTMiPmN3C909rsEWMNqJqhIVPGP3Exg==`)
	sessionStoreKey, _ := base64.StdEncoding.DecodeString(`AbfYwmmt8UCwUuhd9qvfNA9UCuN1cVcKJN1ofbiky6xCyyBj20whe40rJa3Su0WOWLWcPpO1taqJdsEI/65+JA==`)
	cookieStore = securecookie.New(cookieStoreKey, nil)
	sessionStore = sessions.NewCookieStore(sessionStoreKey)
}

func SetupAuthboss() {

	hostEnv := os.Getenv("HOST_NEWGO")
	if hostEnv == "" {
		hostEnv = os.Getenv("HOST_MINIO")
	}
	if hostEnv == "" {
		hostEnv = `http://localhost:9000`
	}

	ab.Storer = database
	ab.OAuth2Storer = database
	ab.MountPath = "/auth"
	ab.ViewsPath = "myauthboss/ab_views"
	ab.RootURL = hostEnv

	ab.LayoutDataMaker = layoutData

	ab.OAuth2Providers = map[string]authboss.OAuth2Provider{
		"google": authboss.OAuth2Provider{
			OAuth2Config: &oauth2.Config{
				ClientID:     ``,
				ClientSecret: ``,
				Scopes:       []string{`profile`, `email`},
				Endpoint:     google.Endpoint,
			},
			Callback: aboauth.Google,
		},
	}

	b, err := ioutil.ReadFile(filepath.Join("myauthboss/views", "layout.html.tpl"))
	if err != nil {
		panic(err)
	}
	ab.Layout = template.Must(template.New("layout").Funcs(funcs).Parse(string(b)))

	ab.XSRFName = "csrf_token"
	ab.XSRFMaker = func(_ http.ResponseWriter, r *http.Request) string {
		return nosurf.Token(r)
	}

	ab.CookieStoreMaker = NewCookieStorer
	ab.SessionStoreMaker = NewSessionStorer

	//ab.Mailer = authboss.LogMailer(os.Stdout)
	// Fetch email password from environment variables if any.
	emailPassword := os.Getenv("NEWGO_EMAIL_PASSWORD")
	ab.Mailer = authboss.SMTPMailer("smtp.gmail.com:587", smtp.PlainAuth("", "reuben.yang@gmail.com", emailPassword, "smtp.gmail.com"))

	ab.Policies = []authboss.Validator{
		authboss.Rules{
			FieldName:       "email",
			Required:        true,
			AllowWhitespace: false,
		},
		authboss.Rules{
			FieldName:       "password",
			Required:        true,
			MinLength:       4,
			MaxLength:       8,
			AllowWhitespace: false,
		},
	}

	ab.RegisterOKPath = "/auth/login"
	ab.AuthLoginOKPath = "/redirectMinio"
	ab.AuthLoginFailPath = "/auth/login"
	ab.AuthLogoutOKPath = "/redirectMinio"

	if err := ab.Init(); err != nil {
		log.Fatal(err)
	}
}

func layoutData(w http.ResponseWriter, r *http.Request) authboss.HTMLData {
	currentUserName := ""
	userInter, err := ab.CurrentUser(w, r)
	if userInter != nil && err == nil {
		currentUserName = userInter.(*User).Name
	}

	return authboss.HTMLData{
		"loggedin":               userInter != nil,
		"username":               "",
		authboss.FlashSuccessKey: ab.FlashSuccess(w, r),
		authboss.FlashErrorKey:   ab.FlashError(w, r),
		"current_user_name":      currentUserName,
	}
}

func RedirectMinio(w http.ResponseWriter, r *http.Request, minioToken string) {
	data := layoutData(w, r).MergeKV("minioToken", minioToken)
	mustRender(w, r, "redirect_minio", data)
}

func mustRender(w http.ResponseWriter, r *http.Request, name string, data authboss.HTMLData) {
	data.MergeKV("csrf_token", nosurf.Token(r))
	err := templates.Render(w, name, data)
	if err == nil {
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, "Error occurred rendering template:", err)
}

func badRequest(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(w, "Bad request:", err)

	return true
}

package users

import (
	".././cookies"
	".././dao"
	".././templating"
	"fmt"
	"net/http"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if cookies.IsLoggedIn(r) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		templating.RenderPage(w, "register", nil)
	case "POST":
		username := r.FormValue("username")
		password := r.FormValue("password")

		accessLevel := r.FormValue("access_level")
		if accessLevel != "" && !dao.IsAdmin(r) {
			fmt.Fprintf(w, `
				<body style="background: black; text-align: center;">
					<video src="/images/gandalf.mp4" autoplay loop>You Shall Not Pass!</video>
				</body>
			`)
			return
		}
		admin := accessLevel == "admin"

		userID, err := Register(username, password, admin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cookies.Login(r, w, userID, username)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		templating.ErrorPage(w, http.StatusMethodNotAllowed)
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if cookies.IsLoggedIn(r) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		templating.RenderPage(w, "login", nil)
	case "POST":
		username := r.FormValue("username")
		password := r.FormValue("password")

		userID, ok := Login(username, password)
		if !ok {
			http.Error(w, "Invalid username or password.", http.StatusBadRequest)
			return
		}

		cookies.Login(r, w, userID, username)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		templating.ErrorPage(w, http.StatusMethodNotAllowed)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		cookies.Logout(r, w)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		templating.ErrorPage(w, http.StatusMethodNotAllowed)
	}
}

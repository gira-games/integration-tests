package server

import (
	"net/http"

	"github.com/asankov/gira/pkg/models"
)

func (s *Server) handleUserSignupForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, &gamesData{}, signupUserPage)
	}
}

func (s *Server) handleUserLoginForm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, &gamesData{}, loginUserPage)
	}
}

func (s *Server) handleUserLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email, password := r.PostFormValue("email"), r.PostFormValue("password")
		res, err := s.Client.LoginUser(&models.User{
			Email:    email,
			Password: password,
		})
		if err != nil {
			s.Log.Printf("error while logging in user: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  "token",
			Value: res.Token,
			Path:  "/",
		})
		w.Header().Add("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	}
}

func (s *Server) handleUserSignup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, email, password := r.PostFormValue("username"), r.PostFormValue("email"), r.PostFormValue("password")

		if _, err := s.Client.CreateUser(&models.User{
			Username: username,
			Email:    email,
			Password: password,
		}); err != nil {
			s.Log.Printf("error while creating user: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: this is not shown
		s.Session.Put(r, "flash", "User created succesfully.")

		w.Header().Add("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	}
}
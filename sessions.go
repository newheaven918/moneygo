package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Session struct {
	SessionId     int64
	SessionSecret string `json:"-"`
	UserId        int64
}

func (s *Session) Write(w http.ResponseWriter) error {
	enc := json.NewEncoder(w)
	return enc.Encode(s)
}

func GetSession(r *http.Request) (*Session, error) {
	var s Session

	cookie, err := r.Cookie("moneygo-session")
	if err != nil {
		return nil, fmt.Errorf("moneygo-session cookie not set")
	}
	s.SessionSecret = cookie.Value

	err = DB.SelectOne(&s, "SELECT * from sessions where SessionSecret=?", s.SessionSecret)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func DeleteSessionIfExists(r *http.Request) {
	session, err := GetSession(r)
	if err == nil {
		DB.Delete(session)
	}
}

func NewSessionCookie() (string, error) {
	bits := make([]byte, 128)
	if _, err := io.ReadFull(rand.Reader, bits); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bits), nil
}

func NewSession(w http.ResponseWriter, r *http.Request, userid int64) (*Session, error) {
	s := Session{}

	session_secret, err := NewSessionCookie()
	if err != nil {
		return nil, err
	}

	cookie := http.Cookie{
		Name:     "moneygo-session",
		Value:    session_secret,
		Path:     "/",
		Domain:   r.URL.Host,
		Expires:  time.Now().AddDate(0, 1, 0), // a month from now
		Secure:   true,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)

	s.SessionSecret = session_secret
	s.UserId = userid

	err = DB.Insert(&s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func SessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" || r.Method == "PUT" {
		user_json := r.PostFormValue("user")
		if user_json == "" {
			WriteError(w, 3 /*Invalid Request*/)
			return
		}

		user := User{}
		err := user.Read(user_json)
		if err != nil {
			WriteError(w, 3 /*Invalid Request*/)
			return
		}

		dbuser, err := GetUserByUsername(user.Username)
		if err != nil {
			WriteError(w, 2 /*Unauthorized Access*/)
			return
		}

		user.HashPassword()
		if user.PasswordHash != dbuser.PasswordHash {
			WriteError(w, 2 /*Unauthorized Access*/)
			return
		}

		DeleteSessionIfExists(r)

		session, err := NewSession(w, r, dbuser.UserId)
		if err != nil {
			WriteError(w, 999 /*Internal Error*/)
			return
		}

		err = session.Write(w)
		if err != nil {
			WriteError(w, 999 /*Internal Error*/)
			log.Print(err)
			return
		}
	} else if r.Method == "GET" {
		s, err := GetSession(r)
		if err != nil {
			WriteError(w, 1 /*Not Signed In*/)
			return
		}

		s.Write(w)
	} else if r.Method == "DELETE" {
		DeleteSessionIfExists(r)
		WriteSuccess(w)
	}
}

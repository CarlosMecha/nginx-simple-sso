package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/adler32"
	"log"
	"net/http"
	"os"
	"strconv"
)

const loginHTML = `
<html>
	<head>
		<title>Login</title>
	</head>
	<body>
		<form action="/login" method="post">
			<input type="text" placeholder="Enter Username" name="username" required/>
			<input type="password" placeholder="Enter Password" name="password" required/>
			<button type="submit">Login</button>	
		</form>
	</body>
</html>
`

const cookieName = "nginx-auth"
const headerName = "X-Auth-Token"
const userHeaderName = "X-Auth-User-ID"

var ErrInvalidCredentials = errors.New("invalid credentials")
var auth AuthService

type AuthService interface {
	Authenticate(string) (int, error)
	GetUser(int) (User, error)
	Login(Credentials) (string, error)
}

type User struct {
	ID       int
	Username string
	Email    string

	password string
}

type Credentials struct {
	username string
	password string
}

type inMemoryAuth struct {
	users map[int]User
}

func (a *inMemoryAuth) Authenticate(token string) (int, error) {
	t, err := strconv.ParseUint(token, 10, 32)
	if err == nil {
		for id, user := range a.users {
			h := adler32.New()
			h.Write([]byte(fmt.Sprintf("%d:%s:%s", id, user.Username, user.password)))
			if uint32(t) == h.Sum32() {
				return id, nil
			}
		}
	}

	return 0, ErrInvalidCredentials
}

func (a *inMemoryAuth) GetUser(id int) (User, error) {
	if user, found := a.users[id]; found {
		return user, nil
	}
	return User{}, errors.New("user not found")
}

func (a *inMemoryAuth) Login(cred Credentials) (string, error) {
	for id, user := range a.users {
		if user.Username == cred.username {
			if user.password == cred.password {
				h := adler32.New()
				h.Write([]byte(fmt.Sprintf("%d:%s:%s", id, user.Username, user.password)))
				return fmt.Sprintf("%d", h.Sum32()), nil
			}
			return "", ErrInvalidCredentials
		}
	}

	return "", ErrInvalidCredentials
}

func writeLoginPage(w http.ResponseWriter) {
	w.Header()["Content-Type"] = []string{"text/html"}
	if _, err := w.Write([]byte(loginHTML)); err != nil {
		log.Fatalf("Unable to write the response: %s\n", err.Error())
		w.WriteHeader(500)
	}
}

func getCredentials(r *http.Request) (Credentials, error) {
	if err := r.ParseForm(); err != nil {
		log.Fatalf("Unable to parse form: %s\n", err.Error())
		return Credentials{}, err
	}

	if r.Form["username"] == nil || r.Form["password"] == nil || len(r.Form["username"]) != 1 || len(r.Form["password"]) != 1 {
		log.Fatalln("Invalid credentials provided")
		return Credentials{}, ErrInvalidCredentials
	}

	return Credentials{
		username: r.Form["username"][0],
		password: r.Form["password"][0],
	}, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request %s %s\n", r.Method, r.URL.String())
	log.Printf("Headers: %+v\n", r.Header)

	switch r.URL.Path {
	case "/login":
		if r.Method == "GET" {
			writeLoginPage(w)
			return
		}

		creds, err := getCredentials(r)
		if err != nil {
			log.Println("Unauthorized request")
			w.WriteHeader(401)
			if _, err := w.Write([]byte("Invalid credentials\n")); err != nil {
				log.Fatalf("Unable to write the response: %s\n", err.Error())
				w.WriteHeader(500)
			}
			return
		}

		token, err := auth.Login(creds)
		if err != nil {
			log.Println("Unauthorized request")
			w.WriteHeader(401)
			if _, err := w.Write([]byte("Invalid credentials\n")); err != nil {
				log.Fatalf("Unable to write the response: %s\n", err.Error())
				w.WriteHeader(500)
			}
			return
		}

		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    token,
			Domain:   r.URL.Host,
			HttpOnly: true,
			// 20 mins
			MaxAge: 20 * 60,
		}

		http.SetCookie(w, cookie)
		w.WriteHeader(200)

	case "/logout":
		cookie, err := r.Cookie(cookieName)
		if err == http.ErrNoCookie {
			w.WriteHeader(200)
			break
		} else if err != nil {
			log.Fatalf("Unable to read cookies: %s\n", err.Error())
			w.WriteHeader(500)
			return
		}

		cookie.Value = "--"
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)
		w.WriteHeader(200)

	case "/auth":
		token, found := r.Header[headerName]
		if !found || len(token) != 1 {
			log.Println("Unauthorized request")
			w.WriteHeader(401)
			return
		}

		userID, err := auth.Authenticate(token[0])
		if err != nil {
			log.Println("Unauthorized request")
			w.WriteHeader(401)
			if _, err := w.Write([]byte("Unauthorized request\n")); err != nil {
				log.Fatalf("Unable to write the response: %s\n", err.Error())
				w.WriteHeader(500)
			}
			return
		}

		log.Println("Authorized request")
		w.Header()[userHeaderName] = []string{fmt.Sprintf("%d", userID)}
		w.WriteHeader(200)

	case "/me":
		cookie, err := r.Cookie(cookieName)
		if err == http.ErrNoCookie {
			log.Println("Unauthorized request")
			w.WriteHeader(401)
			return
		} else if err != nil {
			log.Fatalf("Unable to retrieve cookie: %s\n", err.Error())
			w.WriteHeader(500)
			return
		}

		userID, err := auth.Authenticate(cookie.Value)
		if err != nil {
			log.Println("Unauthorized request")
			w.WriteHeader(401)
			if _, err := w.Write([]byte("Unauthorized request\n")); err != nil {
				log.Fatalf("Unable to write the response: %s\n", err.Error())
				w.WriteHeader(500)
			}
		}

		log.Println("Authorized Me request")
		user, err := auth.GetUser(userID)
		if err != nil {
			log.Fatalf("Unable to retrieve user: %s\n", err.Error())
			w.WriteHeader(500)
			return
		}

		w.Header()["Content-Type"] = []string{"application/json"}
		content, err := json.Marshal(user)
		if err != nil {
			log.Fatalf("Unable to encode the response: %s\n", err.Error())
			w.WriteHeader(500)
			return
		}

		if _, err := w.Write(content); err != nil {
			log.Fatalf("Unable to write the response: %s\n", err.Error())
			w.WriteHeader(500)
			return
		}

	}

	log.Println("Request completed")
}

func main() {
	log.SetOutput(os.Stdout)

	auth = &inMemoryAuth{
		users: map[int]User{
			1: User{
				ID:       1,
				Username: "carlos",
				Email:    "carlos@me.com",
				password: "12345",
			},
			2: User{
				ID:       2,
				Username: "octocat",
				Email:    "octocat@me.com",
				password: "ABCDE",
			},
		},
	}

	http.HandleFunc("/", handler)
	http.ListenAndServe(":80", nil)
}

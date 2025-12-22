package events

import (
	_ "fmt"
	Admin "hifi/Events/Admin"
	Auth "hifi/Events/Auth"
	Search "hifi/Events/Search"
	Social "hifi/Events/Social"
	User "hifi/Events/Users"
	Videos "hifi/Events/Videos"

	"github.com/go-chi/chi/v5"
)

func Init() {
	Videos.View = Social.View
}

func Handler(req chi.Router) {
	req.Route("/auth", Auth.Handle)
	req.Route("/users", User.Handle)
	req.Route("/videos", Videos.Handle)

	req.Route("/social/users", Social.HandleUsers)
	req.Route("/social/videos", Social.HandleVideos)

	req.Route("/admin", Admin.Handle)

	req.Route("/search", Search.Handle)

}

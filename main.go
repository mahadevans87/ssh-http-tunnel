package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"strconv"

	"github.com/gliderlabs/ssh"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

type Tunnel struct {
	w      io.Writer
	doneCh chan struct{}
}

var tunnels = map[int]chan Tunnel{}

func main() {
	go func() {
		SetupHTTPServer()
	}()

	ssh.Handle(func(s ssh.Session) {
		id := rand.Intn(math.MaxInt)
		tunnels[id] = make(chan Tunnel)
		fmt.Println("Tunnel ID -> ", id)
		tunnel := <-tunnels[id]

		fmt.Println("Tunnel setup. Begin copy")

		_, err := io.Copy(tunnel.w, s)
		if err != nil {
			log.Fatalln(err)
		}
		close(tunnel.doneCh)

		s.Write([]byte("\nDone transferring.\n"))
		s.Close()
	})

	log.Fatalln(ssh.ListenAndServe(":2222", nil, ssh.HostKeyFile("id_rsa")))
}

func SetupHTTPServer() {

	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{Views: engine})

	app.Get("/:id", func(ctx *fiber.Ctx) error {
		idstr := ctx.Params("id")
		id, _ := strconv.Atoi(idstr)
		return ctx.Render("file", fiber.Map{
			"Title": "Get a File!",
			"ID":    id,
		})
	})

	app.Get("/:id/raw", func(ctx *fiber.Ctx) error {
		tunnelId, err := strconv.Atoi(ctx.Params("id"))
		if err != nil {
			ctx.Write([]byte("Could not locate tunnel"))
		} else {
			tunnelCh, found := tunnels[tunnelId]
			if !found {
				ctx.Write([]byte("Could not find tunnel with that ID"))
			} else {
				doneCh := make(chan struct{})
				tunnel := Tunnel{
					w:      ctx.Response().BodyWriter(),
					doneCh: doneCh,
				}
				tunnelCh <- tunnel
				<-doneCh

			}
		}
		return nil
	})
	app.Listen(":3000")
}

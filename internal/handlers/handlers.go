package handlers

import (
	"gopkg.in/gorp.v1"
	"log"
	"net/http"
	"path"
	"strings"
)

// But who writes the ResponseWriterWriter?
type ResponseWriterWriter interface {
	Write(http.ResponseWriter) error
}

type Tx = gorp.Transaction

type Context struct {
	Tx           *Tx
	User         *User
	remainingURL string // portion of URL path not yet reached in the hierarchy
}

func (c *Context) SetURL(url string) {
	c.remainingURL = path.Clean("/" + url)[1:]
}

func (c *Context) NextLevel() string {
	split := strings.SplitN(c.remainingURL, "/", 2)
	if len(split) == 2 {
		c.remainingURL = split[1]
	} else {
		c.remainingURL = ""
	}
	return split[0]
}

type Handler func(*http.Request, *Context) ResponseWriterWriter

type APIHandler struct {
	DB *gorp.DbMap
}

func (ah *APIHandler) txWrapper(h Handler, r *http.Request, context *Context) (writer ResponseWriterWriter) {
	tx, err := ah.DB.Begin()
	if err != nil {
		log.Print(err)
		return NewError(999 /*Internal Error*/)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
		if _, ok := writer.(*Error); ok {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Print(err)
				writer = NewError(999 /*Internal Error*/)
			}
		}
	}()

	context.Tx = tx
	return h(r, context)
}

func (ah *APIHandler) route(r *http.Request) ResponseWriterWriter {
	context := &Context{}
	context.SetURL(r.URL.Path)
	if context.NextLevel() != "v1" {
		return NewError(3 /*Invalid Request*/)
	}

	route := context.NextLevel()

	switch route {
	case "sessions":
		return ah.txWrapper(SessionHandler, r, context)
	case "users":
		return ah.txWrapper(UserHandler, r, context)
	case "securities":
		return ah.txWrapper(SecurityHandler, r, context)
	case "securitytemplates":
		return SecurityTemplateHandler(r, context)
	case "prices":
		return ah.txWrapper(PriceHandler, r, context)
	case "accounts":
		return ah.txWrapper(AccountHandler, r, context)
	case "transactions":
		return ah.txWrapper(TransactionHandler, r, context)
	case "imports":
		return ah.txWrapper(ImportHandler, r, context)
	case "reports":
		return ah.txWrapper(ReportHandler, r, context)
	default:
		return NewError(3 /*Invalid Request*/)
	}
}

func (ah *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ah.route(r).Write(w)
}

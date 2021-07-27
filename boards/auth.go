package boards

import (
	"context"
	"fmt"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/goji/httpauth"
)

const jiraClientCtx authCtx = iota

type authCtx int

func (b *Board) jiraBasicAuth() func(next http.Handler) http.Handler {
	bHandler := httpauth.BasicAuth(httpauth.AuthOptions{
		Realm: "Authenticate to Jira",
		AuthFunc: func(user string, password string, req *http.Request) bool {
			tp := jira.BasicAuthTransport{
				Username: user,
				Password: password,
			}

			jiraClient, err := jira.NewClient(tp.Client(), b.jiraEndpoint)
			if err != nil {
				fmt.Println(err.Error())
				return false
			}
			_, _, err = jiraClient.User.GetSelf()
			if err != nil {
				fmt.Println(err.Error())
				return false
			}
			ctxValueReq := req.WithContext(context.WithValue(req.Context(), jiraClientCtx, jiraClient))
			*req = *ctxValueReq
			return true
		},
	})
	return bHandler
}

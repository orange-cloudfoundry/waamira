package boards

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/foolin/goview"
	"github.com/gorilla/mux"
	bfconfluence "github.com/kentaro-m/blackfriday-confluence"
	bf "github.com/russross/blackfriday/v2"

	"github.com/orange-cloudfoundry/waamira/flatten"
	"github.com/orange-cloudfoundry/waamira/front"
)

var renderer = &bfconfluence.Renderer{}

var skipKeyForFlatten = map[string]bool{
	"created":        true,
	"duedate":        true,
	"resolutiondate": true,
	"updated":        true,
}

type Board struct {
	jiraEndpoint  string
	templateFiles map[string]jira.IssueFields
	gv            *goview.ViewEngine
}

func NewBoard(jiraEndpoint string, templateFiles map[string]jira.IssueFields) *Board {
	gv := goview.New(goview.Config{
		Root:      "templates",
		Extension: ".gohtml",
		Master:    "layouts/master",
		Partials:  []string{},
		Funcs: template.FuncMap{
			"title": strings.Title,
		},
		DisableCache: true,
		Delims:       goview.Delims{Left: "{{", Right: "}}"},
	})
	gv.SetFileHandler(func(config goview.Config, tplFile string) (content string, err error) {
		b, err := front.Templates.ReadFile(filepath.Join(config.Root, tplFile) + config.Extension)
		if err != nil {
			return "", err
		}
		return string(b), nil
	})
	return &Board{
		jiraEndpoint:  jiraEndpoint,
		templateFiles: templateFiles,
		gv:            gv,
	}
}

func (b *Board) Index(w http.ResponseWriter, req *http.Request) {

	err := b.gv.Render(w, http.StatusOK, "index", goview.M{
		"TemplateFiles": b.templateFiles,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Render index error: %v!", err), http.StatusInternalServerError)
		return
	}
}

func (b *Board) Template(w http.ResponseWriter, req *http.Request) {
	name := mux.Vars(req)["name"]
	err := b.gv.Render(w, http.StatusOK, "templateform", goview.M{
		"FlattenFields": b.fieldsToFlattenMap(b.templateFiles[name]),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Render index error: %v!", err), http.StatusInternalServerError)
		return
	}
}

func (b *Board) Create(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fields, err := b.decodeMapToIssue(b.valuesToMap(req.PostForm))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ast := bf.
		New(bf.WithRenderer(renderer), bf.WithExtensions(bf.CommonExtensions)).
		Parse([]byte(fields.Description))
	summary := renderer.Render(ast)
	fields.Description = string(summary)
	jiraClient := req.Context().Value(jiraClientCtx).(*jira.Client)

	fmt.Printf("%#v\n", fields)
	issue, resp, err := jiraClient.Issue.Create(&jira.Issue{
		Fields: &fields,
	})
	if err != nil {
		defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("%s, body: \n%s", err.Error(), string(b)), http.StatusInternalServerError)
		return
	}
	err = b.gv.Render(w, http.StatusOK, "create", goview.M{
		"link": b.jiraEndpoint + "/browse/" + issue.Key,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Render index error: %v!", err), http.StatusInternalServerError)
		return
	}
}

func (b *Board) valuesToMap(values url.Values) map[string]interface{} {
	m := make(map[string]interface{})

	for k, v := range values {
		if len(v) == 0 {
			continue
		}
		m[k] = v[0]
	}
	return m
}

func (b *Board) fieldsToFlattenMap(fields jira.IssueFields) map[string]interface{} {
	var m map[string]interface{}
	byt, _ := json.Marshal(fields)
	_ = json.Unmarshal(byt, &m) // nolint
	for k, v := range fields.Unknowns {
		m[k] = v
	}
	delete(m, "Unknowns")
	m = flatten.Flatten(m)
	for k, _ := range skipKeyForFlatten {
		delete(m, k)
	}

	return m
}

func (b *Board) decodeMapToIssue(m map[string]interface{}) (jira.IssueFields, error) {
	byt, err := json.Marshal(flatten.Expand(m))
	if err != nil {
		return jira.IssueFields{}, err
	}
	var fields jira.IssueFields
	err = json.Unmarshal(byt, &fields)
	if err != nil {
		return jira.IssueFields{}, err
	}
	return fields, nil
}

func (b *Board) RegisterRoutes(router *mux.Router) {
	router.Use(b.jiraBasicAuth())
	router.PathPrefix("/static/").Handler(http.FileServer(http.FS(front.Static)))
	router.HandleFunc("/", b.Index)
	router.HandleFunc("/board", b.Index)
	router.HandleFunc("/template/{name:[^/]*}", b.Template).Methods(http.MethodGet)
	router.HandleFunc("/template", b.Create).Methods(http.MethodPost)
}

# Waamira

Create issue with ease on Jira based on templates you've made ! description allow markdown too <3

## Usage 

1. git clone this repo
2. build with `go build .`
3. add a config file named `config.yml` with this content at least:

```yaml
listen: 0.0.0.0:8080
jira:
  endpoint: https://enpoint.jira.com
templates_dir: templates
```
4. Add templates in a folder you've named in config file (here `templates`). Template must be in json format. 
you can see example at [templates](/templates) in this repo. This is actually fields as defined in at https://github.com/andygrunwald/go-jira/blob/e8880eb25076e18451dd39c0060b2b0ae8bcfa89/issue.go#L104-L147
5. run server with `./waamira`
6. access it with your favorite browser at http://localhost:8080 and enjoy !


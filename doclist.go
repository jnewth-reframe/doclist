package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var apiCallCount int

func verify(test bool, format string, va ...any) {
	if !test {
		log.Fatalf(format, va...)
	}
}

func loadSecrets(filename string) (access, secret string) {
	bytes, err := os.ReadFile(filename)
	verify(err == nil, "failed to read %s: %v", filename, err)
	var keys map[string]any
	err = json.Unmarshal(bytes, &keys)
	verify(err == nil, "error parsing JSON in %s: %v", filename, err)
	var ok bool
	access, ok = keys["accessKey"].(string)
	verify(ok, "failed to retrieve accessKey")
	secret, ok = keys["secretKey"].(string)
	verify(ok, "failed to retrieve secretKey")
	return access, secret
}

func apiGetRaw(access, secret, rawURL string) []byte {
	apiCallCount++
	req, err := http.NewRequest("GET", rawURL, nil)
	verify(err == nil, "failed to build request: %v", err)
	req.Header.Set("Accept", "application/json;charset=UTF-8; qs=0.09")
	req.SetBasicAuth(access, secret)
	resp, err := http.DefaultClient.Do(req)
	verify(err == nil, "request failed: %v", err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	verify(err == nil, "failed to read response body: %v", err)
	verify(resp.StatusCode == 200, "API returned status %d\n  url: %s\n  response: %s", resp.StatusCode, rawURL, string(body))
	return body
}

// apiItem is one entry from the getDocuments API response.
type apiItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsContainer bool   `json:"isContainer"`
	ViewRef     string `json:"viewRef"`
}

type apiPage struct {
	Items []apiItem `json:"items"`
	Next  string    `json:"next"`
}

// node is a folder or document in the tree we build locally.
type node struct {
	ID       string
	Name     string
	URL      string
	IsFolder bool
	Children []*node
}

// fetchItems calls GET /api/globaltreenodes/<nodeType>/<id>, following pagination.
// nodeType is "folder" for regular folders or "resourcecompanyowner" for company roots.
// If dumpDir is non-empty, each raw response page is written there as
// <folderID>_page<n>.json for inspection.
func fetchItems(access, secret, nodeType, folderID, dumpDir string) []apiItem {
	nextURL := fmt.Sprintf(
		"https://cad.onshape.com/api/globaltreenodes/%s/%s?limit=20&sortColumn=name&sortOrder=asc",
		nodeType, folderID,
	)
	var all []apiItem
	page := 1
	for nextURL != "" {
		body := apiGetRaw(access, secret, nextURL)
		if dumpDir != "" {
			var raw any
			json.Unmarshal(body, &raw)
			pretty, _ := json.MarshalIndent(raw, "", "  ")
			dumpPath := filepath.Join(dumpDir, fmt.Sprintf("%s_page%d.json", folderID, page))
			err := os.WriteFile(dumpPath, pretty, 0644)
			verify(err == nil, "failed to write dump %s: %v", dumpPath, err)
			fmt.Println("  dump:", dumpPath)
		}
		var p apiPage
		err := json.Unmarshal(body, &p)
		verify(err == nil, "failed to parse response for folder %s: %v", folderID, err)
		all = append(all, p.Items...)
		nextURL = p.Next
		page++
	}
	return all
}

// buildTree recursively fetches folder contents and assembles a node tree.
// nodeType is used for this node's fetch ("folder" or "resourcecompanyowner").
// Children are always fetched as "folder". maxDepth 0 = unlimited.
func buildTree(access, secret, nodeType string, n *node, dumpDir string, maxDepth, currentDepth int) {
	items := fetchItems(access, secret, nodeType, n.ID, dumpDir)
	for _, it := range items {
		url := it.ViewRef
		if url == "" {
			if it.IsContainer {
				url = "https://cad.onshape.com/documents?nodeId=" + it.ID + "&resourceType=folder"
			} else {
				url = "https://cad.onshape.com/documents/" + it.ID
			}
		}
		child := &node{
			ID:       it.ID,
			Name:     it.Name,
			URL:      url,
			IsFolder: it.IsContainer,
		}
		if it.IsContainer && (maxDepth == 0 || currentDepth < maxDepth) {
			buildTree(access, secret, "folder", child, dumpDir, maxDepth, currentDepth+1)
		}
		n.Children = append(n.Children, child)
	}
	sort.Slice(n.Children, func(i, j int) bool {
		ci, cj := n.Children[i], n.Children[j]
		if ci.IsFolder != cj.IsFolder {
			return ci.IsFolder // folders before documents
		}
		return ci.Name < cj.Name
	})
}

// --- HTML output ---

func writeHTMLNodes(sb *strings.Builder, nodes []*node, depth int) {
	indent := strings.Repeat("  ", depth)
	sb.WriteString(indent + "<ul>\n")
	for _, n := range nodes {
		if n.IsFolder {
			sb.WriteString(indent + "  <li class=\"folder\">\n")
			sb.WriteString(indent + "    <span class=\"arrow\" onclick=\"toggle(this)\">&#9660;</span>\n")
			if n.URL != "" {
				fmt.Fprintf(sb, indent+`    <a href="%s">%s</a>`+"\n", html.EscapeString(n.URL), html.EscapeString(n.Name))
			} else {
				sb.WriteString(indent + "    " + html.EscapeString(n.Name) + "\n")
			}
			writeHTMLNodes(sb, n.Children, depth+2)
			sb.WriteString(indent + "  </li>\n")
		} else {
			sb.WriteString(indent + "  <li>")
			if n.URL != "" {
				fmt.Fprintf(sb, `<a href="%s">%s</a>`, html.EscapeString(n.URL), html.EscapeString(n.Name))
			} else {
				sb.WriteString(html.EscapeString(n.Name))
			}
			fmt.Fprintf(sb, ` <span class="docid">%s</span>`, html.EscapeString(n.ID))
			sb.WriteString("</li>\n")
		}
	}
	sb.WriteString(indent + "</ul>\n")
}

func writeHTML(root *node, filename string) {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>` + html.EscapeString(root.Name) + `</title>
  <style>
    body { font-family: sans-serif; font-size: 14px; margin: 2em; }
    ul { list-style: none; padding-left: 1.5em; }
    li { margin: 0.2em 0; }
    a { text-decoration: none; color: #1a6eb5; }
    a:hover { text-decoration: underline; }
    li > ul { border-left: 1px solid #ccc; margin-left: 0.4em; padding-left: 1em; }
    .docid { font-size: 0.8em; color: #888; margin-left: 0.5em; font-family: monospace; }
    .arrow { cursor: pointer; user-select: none; display: inline-block; width: 1em; font-size: 0.8em; color: #555; }
    .arrow.collapsed { transform: rotate(-90deg); }
    li.folder > ul.collapsed { display: none; }
  </style>
</head>
<body>
  <h1>`)
	if root.URL != "" {
		fmt.Fprintf(&sb, `<a href="%s">%s</a>`, html.EscapeString(root.URL), html.EscapeString(root.Name))
	} else {
		sb.WriteString(html.EscapeString(root.Name))
	}
	sb.WriteString("</h1>\n")
	writeHTMLNodes(&sb, root.Children, 1)
	sb.WriteString(`<script>
function toggle(arrow) {
  var ul = arrow.parentElement.querySelector('ul');
  if (!ul) return;
  var collapsed = ul.classList.toggle('collapsed');
  arrow.classList.toggle('collapsed', collapsed);
}
</script>
</body>
</html>
`)

	err := os.WriteFile(filename, []byte(sb.String()), 0644)
	verify(err == nil, "failed to write %s: %v", filename, err)
	fmt.Println("+", filename)
}

// --- Graphviz DOT output ---

func writeDOTNodes(sb *strings.Builder, n *node) {
	shape := "note"
	if n.IsFolder {
		shape = "folder"
	}
	fmt.Fprintf(sb, "  %q [label=%q, shape=%s, URL=%q];\n", n.ID, n.Name, shape, n.URL)
	for _, child := range n.Children {
		fmt.Fprintf(sb, "  %q -> %q;\n", n.ID, child.ID)
		writeDOTNodes(sb, child)
	}
}

func writeDOT(root *node, filename string) {
	var sb strings.Builder
	sb.WriteString("digraph doclist {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [fontname=\"sans-serif\"];\n")
	writeDOTNodes(&sb, root)
	sb.WriteString("}\n")

	err := os.WriteFile(filename, []byte(sb.String()), 0644)
	verify(err == nil, "failed to write %s: %v", filename, err)
	fmt.Println("+", filename)
}

// ---

func main() {
	dump := flag.Bool("dump", false, "write raw JSON responses to <output-base>_dump/")
	depth := flag.Int("depth", 0, "max folder recursion depth (default 0 = unlimited)")
	secretsFile := flag.String("secrets", "secrets.json", "path to secrets.json credentials file")
	rootType := flag.String("root-type", "folder", "globaltreenodes type for root node: folder or resourcecompanyowner")
	flag.Parse()
	args := flag.Args()
	verify(len(args) >= 1, "Usage: doclist [--dump] <folder-id> [output-base]\n  e.g.: doclist 7bff278ed5b795a1f074acb5 reframe")

	folderID := strings.TrimSpace(args[0])
	outputBase := "doclist"
	if len(args) >= 2 {
		outputBase = args[1]
	}

	wd, err := os.Getwd()
	verify(err == nil, "unable to get working directory: %v", err)
	sf := *secretsFile
	if !filepath.IsAbs(sf) {
		sf = filepath.Join(wd, sf)
	}
	access, secret := loadSecrets(sf)

	var dumpDir string
	if *dump {
		dumpDir = outputBase + "_dump"
		err := os.MkdirAll(dumpDir, 0755)
		verify(err == nil, "failed to create dump dir %s: %v", dumpDir, err)
	}

	root := &node{
		ID:       folderID,
		Name:     folderID,
		URL:      fmt.Sprintf("https://cad.onshape.com/documents?nodeId=%s&resourceType=folder", folderID),
		IsFolder: true,
	}

	fmt.Println("building tree...")
	buildTree(access, secret, *rootType, root, dumpDir, *depth, 1)

	writeHTML(root, outputBase+".html")
	writeDOT(root, outputBase+".dot")
	fmt.Printf("api calls: %d\n", apiCallCount)
}

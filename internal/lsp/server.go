package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/kiro-lang/kiro/internal/format"
)

type Server struct {
	docs map[string]*DocState
}

func NewServer() *Server {
	return &Server{docs: map[string]*DocState{}}
}

func (s *Server) Serve(in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	for {
		payload, err := readMessage(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req Request
		if err := json.Unmarshal(payload, &req); err != nil {
			continue
		}
		responses := s.handle(req)
		for _, resp := range responses {
			if err := writeMessage(out, resp); err != nil {
				return err
			}
		}
	}
}

func (s *Server) handle(req Request) []Response {
	switch req.Method {
	case "initialize":
		return []Response{{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"capabilities": map[string]any{
			"textDocumentSync":           1,
			"hoverProvider":              true,
			"definitionProvider":         true,
			"documentFormattingProvider": true,
			"documentSymbolProvider":     true,
			"completionProvider":         map[string]any{"triggerCharacters": []string{"."}},
		}}}}
	case "initialized":
		return nil
	case "shutdown":
		return []Response{{JSONRPC: "2.0", ID: req.ID, Result: nil}}
	case "exit":
		return nil
	case "textDocument/didOpen":
		var p struct{ TextDocument struct{ URI, Text string } }
		_ = json.Unmarshal(req.Params, &p)
		return s.upsertDocument(p.TextDocument.URI, p.TextDocument.Text)
	case "textDocument/didChange":
		var p struct {
			TextDocument   struct{ URI string }
			ContentChanges []struct{ Text string }
		}
		_ = json.Unmarshal(req.Params, &p)
		if len(p.ContentChanges) == 0 {
			return nil
		}
		return s.upsertDocument(p.TextDocument.URI, p.ContentChanges[len(p.ContentChanges)-1].Text)
	case "textDocument/didClose":
		var p struct{ TextDocument struct{ URI string } }
		_ = json.Unmarshal(req.Params, &p)
		delete(s.docs, p.TextDocument.URI)
		return []Response{{JSONRPC: "2.0", Method: "textDocument/publishDiagnostics", Params: map[string]any{"uri": p.TextDocument.URI, "diagnostics": []Diagnostic{}}}}
	case "textDocument/hover":
		var p struct {
			TextDocument struct{ URI string }
			Position     Position
		}
		_ = json.Unmarshal(req.Params, &p)
		return []Response{{JSONRPC: "2.0", ID: req.ID, Result: s.hover(p.TextDocument.URI, p.Position)}}
	case "textDocument/definition":
		var p struct {
			TextDocument struct{ URI string }
			Position     Position
		}
		_ = json.Unmarshal(req.Params, &p)
		return []Response{{JSONRPC: "2.0", ID: req.ID, Result: s.definition(p.TextDocument.URI, p.Position)}}
	case "textDocument/documentSymbol":
		var p struct{ TextDocument struct{ URI string } }
		_ = json.Unmarshal(req.Params, &p)
		d := s.docs[p.TextDocument.URI]
		if d == nil {
			return []Response{{JSONRPC: "2.0", ID: req.ID, Result: []DocumentSymbol{}}}
		}
		return []Response{{JSONRPC: "2.0", ID: req.ID, Result: d.SymbolItems}}
	case "textDocument/formatting":
		var p struct{ TextDocument struct{ URI string } }
		_ = json.Unmarshal(req.Params, &p)
		return []Response{{JSONRPC: "2.0", ID: req.ID, Result: s.formatting(p.TextDocument.URI)}}
	case "textDocument/completion":
		var p struct{ TextDocument struct{ URI string } }
		_ = json.Unmarshal(req.Params, &p)
		return []Response{{JSONRPC: "2.0", ID: req.ID, Result: s.completion(p.TextDocument.URI)}}
	default:
		if req.ID != nil {
			return []Response{{JSONRPC: "2.0", ID: req.ID, Error: &RespError{Code: -32601, Message: "method not found"}}}
		}
		return nil
	}
}

func (s *Server) upsertDocument(uri, text string) []Response {
	doc, diags := buildDocState(uri, text)
	s.docs[uri] = doc
	return []Response{{JSONRPC: "2.0", Method: "textDocument/publishDiagnostics", Params: map[string]any{"uri": uri, "diagnostics": diags}}}
}

func (s *Server) hover(uri string, pos Position) any {
	d := s.docs[uri]
	if d == nil {
		return nil
	}
	word := wordAt(d.Text, pos)
	sym, ok := d.Symbols[word]
	if !ok {
		return nil
	}
	content := sym.Sig
	if sym.Doc != "" {
		content += "\n\n" + sym.Doc
	}
	return map[string]any{"contents": map[string]any{"kind": "markdown", "value": "```kiro\n" + content + "\n```"}, "range": sym.Range}
}

func (s *Server) definition(uri string, pos Position) any {
	d := s.docs[uri]
	if d == nil {
		return nil
	}
	word := wordAt(d.Text, pos)
	sym, ok := d.Symbols[word]
	if !ok {
		return nil
	}
	return []Location{{URI: uri, Range: sym.Range}}
}

func (s *Server) formatting(uri string) any {
	d := s.docs[uri]
	if d == nil {
		return []any{}
	}
	out, err := format.Source(d.Text)
	if err != nil {
		return []any{}
	}
	lines := strings.Split(d.Text, "\n")
	endLine := len(lines) - 1
	endChar := 0
	if len(lines) > 0 {
		endChar = len([]rune(lines[endLine]))
	}
	return []map[string]any{{"range": Range{Start: Position{}, End: Position{Line: endLine, Character: endChar}}, "newText": out}}
}

func (s *Server) completion(uri string) any {
	d := s.docs[uri]
	if d == nil {
		return []CompletionItem{}
	}
	items := []CompletionItem{}
	for _, s := range d.Symbols {
		items = append(items, CompletionItem{Label: s.Name, Kind: s.Kind, Detail: s.Sig})
	}
	for _, kw := range []string{"fn", "type", "const", "if", "when", "for", "while", "spawn", "await", "group", "defer"} {
		items = append(items, CompletionItem{Label: kw, Kind: 14})
	}
	return items
}

func wordAt(src string, pos Position) string {
	lines := strings.Split(src, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	r := []rune(lines[pos.Line])
	if pos.Character < 0 || pos.Character >= len(r) {
		return ""
	}
	start, end := pos.Character, pos.Character
	isWord := func(ch rune) bool {
		return ch == '_' || ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
	}
	for start > 0 && isWord(r[start-1]) {
		start--
	}
	for end < len(r) && isWord(r[end]) {
		end++
	}
	if start == end {
		return ""
	}
	return string(r[start:end])
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	length := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			v = strings.TrimSpace(strings.TrimPrefix(v, "content-length:"))
			length, _ = strconv.Atoi(v)
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("invalid content length")
	}
	buf := make([]byte, length)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func writeMessage(w io.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body)))
	b.Write(body)
	_, err = w.Write(b.Bytes())
	return err
}

func PathFromURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	if u.Scheme != "file" {
		return uri
	}
	return u.Path
}

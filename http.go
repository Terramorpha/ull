package http

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	ipfs "github.com/ipfs/go-ipfs-api"
)

type Node struct {
	Items string  `json:"items"`
	Next  *string `json:"next"`
}

type Item struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func toJson(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (n Node) json() string {
	return toJson(n)
}

//173.178.130.146
var YourIp = "/dns/terramorpha.tech/tcp/4001"

const kind = "dag-cbor"

func LinkedList(sh *ipfs.Shell, lastHash string) func(w http.ResponseWriter, r *http.Request) {
	m := sync.Mutex{}
	var top *string = nil
	topfile, err := os.Open(lastHash)
	if err == nil {
		c, err := ioutil.ReadAll(topfile)
		if err != nil {
			panic(err)
		}
		s := string(c)
		top = &s
	}
	topfile.Close()

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			m.Lock()
			defer m.Unlock()

			id, err := sh.ID()
			if err != nil {
				panic(err)
			}
			addr := fmt.Sprintf("%s/ipfs/%s", YourIp, id.ID)

			enc := json.NewEncoder(w)
			enc.Encode(struct {
				Hash *string `json:"hash"`
				Addr *string `json:"address"`
			}{Hash: top, Addr: &addr})

		case "POST":
			m.Lock()
			defer m.Unlock()
			dec := json.NewDecoder(r.Body)
			content := []Item{}
			dec.Decode(&content)
			if len(content) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			//put in dag the list
			itemsHash, err := sh.DagPut(toJson(content), "json", kind)
			if err != nil {
				panic(err)
			}

			//fmt.Printf("content: %#v\n", content)
			o := Node{
				Items: itemsHash, Next: top,
			}
			h, err := sh.DagPut(o.json(), "json", kind)
			if err != nil {
				panic(err)
			}
			fmt.Println("hash:", h)
			f, err := os.Create(lastHash)
			if err != nil {
				return
			}

			io.WriteString(f, h)
			f.Close()
			top = &h

			enc := json.NewEncoder(w)
			enc.Encode(struct {
				Hash *string `json:"hash"`
			}{Hash: &h})
		default:
			http.NotFound(w, r)
		}
	}
}

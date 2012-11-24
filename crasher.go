package main

import (
	"log"
	"net/http"
	"net/url"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"io"
	"crypto/rand"
)

type Crash struct {
	ProductName string
	Version string
	Dump []byte
	CrashID string
}

func (c *Crash) saveMeta() error {
	filename := "crashes/" + c.CrashID + ".json"
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0600)
}

func (c *Crash) saveDump() error {
	filename := "crashes/" + c.CrashID + ".dump"
	return ioutil.WriteFile(filename, c.Dump, 0600)
}

func crashHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(4096)
	log.Println("Incoming crash")
	crash := makecrash(r.Form)
	log.Println("Crash collected: ", crash.CrashID)
	crash.saveMeta()
	log.Println("Crash metdata saved: ", crash.CrashID)
	crash.saveDump()
	log.Println("Crash dump saved: ", crash.CrashID)
}

func makecrash(form url.Values) Crash {
	return Crash {
		ProductName: form.Get("ProductName"),
		Version: form.Get("Version"),
		Dump: []byte(form.Get("dump")),
		CrashID: makecrashid(),
	}
}

func makecrashid() string {
	return uuid()
}

func uuid() string {
        b := make([]byte, 16)
        _, err := io.ReadFull(rand.Reader, b)
        if err != nil {
                log.Fatal(err)
        }
        b[6] = (b[6] & 0x0F) | 0x40
        b[8] = (b[8] &^ 0x40) | 0x80
        return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func main() {
	http.HandleFunc("/submit", crashHandler)
	log.Println("Listening on port 8080")
        http.ListenAndServe(":8080", nil)
}

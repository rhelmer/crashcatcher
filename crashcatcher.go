package main

import (
	"log"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"io"
	"crypto/rand"
	"os/exec"
)

var rawcrashdir = "./crashes/raw"
var processedcrashdir = "./crashes/processed"

type Crash struct {
	ProductName string
	Version string
	CrashID string
}

func (c *Crash) saveMeta() error {
	filename := rawcrashdir + "/" + c.CrashID + ".json"
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0600)
}

func (c *Crash) process() ([]byte, error) {
	var path = "./build/breakpad/bin/minidump_stackwalk"
	out, err := exec.Command(path, "-m",
		rawcrashdir + "/" + c.CrashID + ".dump").Output()
	return out, err
}

func crashHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Incoming crash")
	var file, _, err = r.FormFile("upload_file_minidump")
	if err != nil {
		fmt.Println(err)
	}
	dumpfile, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	crash := Crash {
		ProductName: r.FormValue("ProductName"),
		Version: r.FormValue("Version"),
		CrashID: makecrashid(),
	}
	log.Println("Crash collected: ", crash.CrashID, crash)
	crash.saveMeta()
	log.Println("Crash metdata saved: ", crash.CrashID)
	filename := rawcrashdir + "/" + crash.CrashID + ".dump"
	ioutil.WriteFile(filename, dumpfile, 0600)
	log.Println("Crash dump saved: ", crash.CrashID)
	var out, _ = crash.process()
	processedfilename := processedcrashdir + "/" + crash.CrashID + ".txt"
	ioutil.WriteFile(processedfilename, out, 0600)
	log.Println("Crash processed: ", crash.CrashID)
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
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

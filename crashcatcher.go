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
var mdswpath = "./build/breakpad/bin/minidump_stackwalk"

type Crash struct {
	ProductName string
	Version string
	CrashID string
	Minidump []byte
}

func (c *Crash) saveMeta() error {
	filename := rawcrashdir + "/" + c.CrashID + ".json"
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0600)
}

func (c *Crash) saveDump() error {
	filename := rawcrashdir + "/" + c.CrashID + ".dump"
	return ioutil.WriteFile(filename, c.Minidump, 0600)
}

func (c *Crash) process() error {
	out, err := exec.Command(mdswpath, "-m",
		rawcrashdir + "/" + c.CrashID + ".dump").Output()
	if err != nil {
		log.Println(err)
	}
	processedfilename := processedcrashdir + "/" + c.CrashID + ".txt"
	return ioutil.WriteFile(processedfilename, out, 0600)
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
		CrashID: MakeCrashID(),
		Minidump: dumpfile,
	}
	log.Println("Crash received: ", crash.CrashID)
	crash.saveMeta()
	log.Println("Crash metadata saved: ", crash.CrashID)
	crash.saveDump()
	log.Println("Crash dump saved: ", crash.CrashID)
	crash.process()
	log.Println("Crash processed: ", crash.CrashID)
}

func MakeCrashID() string {
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

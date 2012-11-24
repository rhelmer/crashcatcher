package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

var rawcrashdir = "./crashdata/raw"
var processedcrashdir = "./crashdata/processed"
var mdswpath = "./build/breakpad/bin/minidump_stackwalk"

// number of cores available for processing
var maxprocs = 1

type Crash struct {
	ProductName string
	Version     string
	CrashID     string
	Minidump    []byte
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

var procsem = make(chan int, maxprocs)

func (c *Crash) process() {
	procsem <- 1
	out, err := exec.Command(mdswpath, "-m",
		rawcrashdir+"/"+c.CrashID+".dump").Output()

	if err != nil {
		log.Println("ERROR during processing of", c.CrashID, err)
	}
	processedfilename := processedcrashdir + "/" + c.CrashID + ".txt"
	err = ioutil.WriteFile(processedfilename, out, 0600)
	if err != nil {
		log.Println("ERROR could not save processed crash", c.CrashID,
			err)
	}
	log.Println("Crash processed and saved:", c.CrashID)
	<-procsem
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
	crash := Crash{
		ProductName: r.FormValue("ProductName"),
		Version:     r.FormValue("Version"),
		CrashID:     MakeCrashID(),
		Minidump:    dumpfile,
	}
	log.Println("Crash received: ", crash.CrashID)
	if err := crash.saveMeta(); err != nil {
		log.Fatal("ERROR could not save crash metadata:",
			crash.CrashID, err)
	} else {
		log.Println("Crash metadata saved: ", crash.CrashID)
	}
	if err := crash.saveDump(); err != nil {
		log.Fatal("ERROR could not save crash dump:",
			crash.CrashID, err)
	} else {
		log.Println("Crash dump saved:", crash.CrashID)
	}
	go crash.process()
	log.Println("Crash dump sent to processor:", crash.CrashID)
}

func MakeCrashID() string {
	return uuid()
}

func uuid() string {
	b := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		log.Fatal("Could not generate UUID", err)
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

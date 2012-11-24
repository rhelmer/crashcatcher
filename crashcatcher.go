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

func saveMeta(crashid string, crashmeta map[string] string) error {
	filename := rawcrashdir + "/" + crashid + ".json"
	b, err := json.Marshal(crashmeta)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0600)
}

func saveDump(crashid string, minidump []byte) error {
	filename := rawcrashdir + "/" + crashid + ".dump"
	return ioutil.WriteFile(filename, minidump, 0600)
}

var procsem = make(chan int, maxprocs)

func process(crashid string, minidump []byte) {
	procsem <- 1
	out, err := exec.Command(mdswpath, "-m",
		rawcrashdir+"/"+crashid+".dump").Output()

	if err != nil {
		log.Println("ERROR during processing of", crashid, err)
	}
	processedfilename := processedcrashdir + "/" + crashid + ".txt"
	err = ioutil.WriteFile(processedfilename, out, 0600)
	if err != nil {
		log.Println("ERROR could not save processed crash", crashid,
			err)
	}
	log.Println("Crash processed and saved:", crashid)
	<-procsem
}

func crashHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Incoming crash")
	var file, _, err = r.FormFile("upload_file_minidump")
	if err != nil {
		fmt.Println(err)
	}
	minidump, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	crashid := MakeCrashID()
	crashmeta := map[string] string {
		"ProductName": r.FormValue("ProductName"),
		"Version":     r.FormValue("Version"),
	}
	log.Println("Crash received: ", crashid)
	if err := saveMeta(crashid, crashmeta); err != nil {
		log.Fatal("ERROR could not save crash metadata:",
			crashid, err)
	} else {
		log.Println("Crash metadata saved: ", crashid)
	}
	if err := saveDump(crashid, minidump); err != nil {
		log.Fatal("ERROR could not save crash dump:",
			crashid, err)
	} else {
		log.Println("Crash dump saved:", crashid)
	}
	go process(crashid, minidump)
	log.Println("Crash dump sent to processor:", crashid)
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

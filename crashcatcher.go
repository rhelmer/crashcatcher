/*
	crashcatcher is a server for collecting and processing crashes
	in minidump format from the google breakpad client:
	http://code.google.com/p/google-breakpad/
*/
package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const (
	// the minidump_stackwalk binary extracts information from minidumps
	mdswpath = "./build/breakpad/bin/minidump_stackwalk"

	// number of cores available for processing
	maxprocs = 1

	// base directory for crash data
	basecrashdir = "./crashdata"
)

var processOnly *bool = flag.Bool("process-only", false,
	"do not run HTTP server, process pending crashes only")
var collectOnly *bool = flag.Bool("collect-only", false,
	"run HTTP server and collect crashes, but do not process")

type Crash struct {
	CrashID	string
	Meta	map[string][]string
	Dump	[]byte
}

// TODO use hashed directory structure
func crashdir(name string, crashid string, extension string) string {
	dir := basecrashdir
	switch(name) {
		// collected crashes are stored here first
		case "incoming":
			dir = dir + "/incoming"
		// after processing, collected crashes are moved here
		case "raw":
			dir = dir + "/raw"
		// output from processing is stored here
		case "processed":
			dir = dir + "/processed"
		default:
			log.Fatal("Crash dir not recognized:",name)
	}
	crashfile := dir + "/" + crashid + "." + extension
	return crashfile
}

// metadata received as key/value pairs is converted to JSON and stored
func (c *Crash) saveMeta() error {
	filename := crashdir("incoming", c.CrashID, "json")
	b, err := json.Marshal(c.Meta)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0600)
}

// minidump files are saved as-is
func (c *Crash) saveDump() error {
	filename := crashdir("incoming", c.CrashID, "dump")
	return ioutil.WriteFile(filename, c.Dump, 0600)
}

// semaphore to limit number of processes per instance
var procsem = make(chan int, maxprocs)

var wg sync.WaitGroup

// minidump_stackwalk prints pipe-delimited data on stdout.
// this is expected to be called as a goroutine, and uses procsem to limit 
// concurrent processors.
func (c *Crash) process() {
	procsem <- 1
	log.Println("start processing", c.CrashID)
	incomingjsonfilename := crashdir("incoming", c.CrashID, "json")
	incomingdumpfilename := crashdir("incoming", c.CrashID, "dump")
	out, err := exec.Command(mdswpath, "-m", incomingdumpfilename).Output()
	if err != nil {
		log.Println("ERROR during processing of", c.CrashID, err)
	}
	processedfilename := crashdir("processed", c.CrashID, "txt")
	err = ioutil.WriteFile(processedfilename, out, 0600)
	if err != nil {
		log.Println("ERROR could not save processed crash", c.CrashID,
			err)
	} else {
		log.Println("Crash processed and saved:", c.CrashID)
		log.Println("Crash raw archived:", c.CrashID)
		err = os.Rename(incomingjsonfilename,
			crashdir("raw", c.CrashID, "json"))
		if err != nil {
			log.Println("ERROR could archive JSON", c.CrashID, err)
		}
		err = os.Rename(incomingdumpfilename,
			crashdir("raw", c.CrashID, "dump"))
		if err != nil {
			log.Println("ERROR could archive dump", c.CrashID, err)
		}
	}
	<-procsem
	if *processOnly {
		wg.Done()
	}
}

// handle "/submit" URLs, expect a mutlipart form with a few required fields
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
	log.Println(r.Form)
	crashmeta := map[string][]string{}
	for k,v :=  range r.Form {
		crashmeta[k] = v
	}
	crash := Crash{
		CrashID: crashid,
		Meta: crashmeta,
		Dump: minidump,
	}
	log.Println("Crash received: ", crashid)
	if err := crash.saveMeta(); err != nil {
		log.Fatal("ERROR could not save crash metadata:",
			crashid, err)
	} else {
		log.Println("Crash metadata saved: ", crashid)
	}
	if err := crash.saveDump(); err != nil {
		log.Fatal("ERROR could not save crash dump:",
			crashid, err)
	} else {
		log.Println("Crash dump saved:", crashid)
	}
	if *collectOnly {
		log.Println("Collect-only mode, not processing:", crashid)
	} else {
		go crash.process()
		log.Println("Crash dump sent to processor:", crashid)
	}
	fmt.Fprintf(w, "CrashID=bp-%v", crashid)
}

// TODO drop date onto last 4 digits of UUID
func MakeCrashID() string {
	return uuid()
}

// from this thread:
// https://groups.google.com/d/topic/golang-nuts/d0nF_k4dSx4/discussion
func uuid() string {
	b := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		log.Fatal("Could not generate UUID", err)
	}
	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] &^ 0x40) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func visit(path string, f os.FileInfo, err error) error {
	extension := ".dump"
	if filepath.Ext(f.Name()) == extension {
		filename := filepath.Base(f.Name())
		basename := filename[:len(filename)-len(extension)]
		log.Println("found dump:", basename)
		crashid := basename
		file, err := os.Open(path)
		if err != nil {
			log.Println(err)
		}
		minidump, err := ioutil.ReadAll(file)
		if err != nil {
			log.Println(err)
		}
		wg.Add(1)
		crash := Crash{
			CrashID: crashid,
			Dump: minidump,
		}
		go crash.process()
	}
	return nil
}

func main() {
	flag.Parse()
	if *processOnly == true {
		log.Println("processing pending crashes")
		err := filepath.Walk(basecrashdir + "/incoming", visit)
		if err != nil {
			log.Println(err)
		}
		wg.Wait()
		close(procsem)
	} else {
		if *collectOnly {
			log.Println("Collect-only mode")
		}
		http.HandleFunc("/submit", crashHandler)
		log.Println("Listening on port 8080")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}
}

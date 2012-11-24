package crashcatcher

import (
	"testing"
	"regexp"
)

func TestCrash(t *testing.T) {
	crash := &Crash{
		ProductName: "WaterWolf",
		Version: "1.2.3",
		Minidump: []byte("abcd"),
	}
	t.Log(crash)
	// TODO mock saveMeta
	// TODO mock saveDump
	// TODO mock process
}

func TestMakeCrashID(t *testing.T) {
        var crashidvalidator = regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")

	var crashid = MakeCrashID()
        if !crashidvalidator.MatchString(crashid) {
		t.Error("crashid does not pass regex:", crashid)
	}
}


package cmd

import (
	"github.com/Unknwon/com"
	"os"
	"os/exec"
	"testing"
)

func TestConflicts(t *testing.T) {
	var err error
	_, err = exec.Command("gopm", "get", "-l", "-r").Output()
	_, err = exec.Command("gopm", "get", "-g", "-r").Output()
	_, err = exec.Command("gopm", "get", "-g", "-l").Output()
	if err == nil {
		t.Fatal("cannot ignore conflicts flags")
	}
}
func TestGetAndRun(t *testing.T) {
	os.Chdir("testproject")
	defer func() {
		os.RemoveAll("src/github.com")
		os.Remove("bin")
		os.Remove("pkg")
		os.Remove(".gopmfile")
		os.Chdir("..")
	}()
	_, err := exec.Command("gopm", "gen", "-l").Output()
	if err != nil {
		t.Log(err)
	}
	if !com.IsDir("bin") || !com.IsDir("pkg") {
		t.Fatal("Gen bin and pkg directories failed.")
	}
	_, err = exec.Command("gopm", "get", "-l").Output()
	if !com.IsDir("src/github.com") {
		t.Fatal("Get packages failed")
	}
	out, err := exec.Command("gopm", "run", "-l").Output()
	if err != nil || string(out) != "TEST\n" {
		t.Fatal("Run failed \t", err.Error())
	}
}

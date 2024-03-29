package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	// "golang.org/x/tools/go/analysis/passes/nilfunc"
)

const (
  semVer = "0.1.0"
  progName = "fiwatcherd"
)


var flagVersion bool
// var flagVerbose bool
// var flagDebug bool
var flagTickrate int
var argFilename string

//setupCliArgs wraps the various commandline arguments and options parsing
//and set up tasks for this program. It will also initiate the argparser 
//and handle basic housekeeping tasks like counting positional arguments 
//and handling arguments such as verson or help
func setupCliArgs () {
  // set up all commandline flags
  // flag.BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")  
  flag.BoolVarP(&flagVersion, "version", "V", false, "print version")  
  // flag.BoolVarP(&flagDebug, "debug", "D", false, "debug output")  
  flag.IntVarP(&flagTickrate, "tickrate", "T", 1000, "service tickrate in millseconds")  
  flag.Parse()

  // if -v/--version is given, print version info and exit
  if flagVersion {
    fmt.Println("v", semVer)
    os.Exit(1)
  }

  // make sure that an incorrect number of args wasn't provided
  expectedArgs := 1
  if len(flag.Args()) != expectedArgs {
    flag.Usage()
    os.Exit(2)
  } else {
    argFilename = flag.Arg(0)
  }
}

// setupLogger wraps the various logger setup tasks for this program
func setupLogger () {
  log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
  log.SetPrefix(progName+": ")
}

func init() {
  setupCliArgs()
  setupLogger()
}

func main() { 
  // setup the ticker for the daemon
  delay := time.Duration(flagTickrate * int(time.Millisecond))
  ticker := time.NewTicker(delay)
  defer ticker.Stop()

  // main daemon loop
  lastcontent := ""
  firstrun := true
  for {
    select {
    case <-ticker.C:
      // try to stat the file
      fi, err := os.Stat(argFilename)
      if err != nil {
        slog.Error("stat error", "err", err)
        continue
      }
      contents := ""
      fc, err := os.ReadFile(argFilename)
      if err != nil {
        slog.Error("read error", "err", err)
        contents = ""
      } else {
        contents = string(fc)
      }
      touchResult := "ok"
      if err := touch(argFilename); err != nil {
        slog.Error("touch error", "err", err)
        touchResult = "failed"
      }
      if !firstrun && lastcontent != contents {
        slog.Warn("content changed", "last", lastcontent, "now", contents)
      }
      slog.Info("fileinfo:", "name", fi.Name(), "size", fi.Size(), "mode", fi.Mode(), "touch", touchResult,  "content", contents )
      lastcontent = contents
      firstrun = false
    }
  }
}

func touch(filePath string) error {
	_, err := os.OpenFile(filePath, os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	currentTime := time.Now().Local()
	return os.Chtimes(filePath, currentTime, currentTime)
}
//
// func main() {
// 	fileName := "temp.txt"
// 	fmt.Println(touch(fileName))
// }

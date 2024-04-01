package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	// "path/filepath"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
)

const (
  semVer = "0.2.2"
  progName = "fiwatcherd"
)

// TODO: add systemd package to provide systemd ready messages 
// TODO: attempt to write PID file 
//       and implement PID file checking logics
// TODO: allow arbitrary levels of concurrency when testing
//       multiple files 

var flagVersion bool
// var flagVerbose bool  //TODO: quiet INFO logs
var flagDebug bool
var flagFix bool
var flagIncrement  bool
// var flagIncAmount int 
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
  flag.BoolVarP(&flagFix, "fix", "F", false, "revert file if truncated")  
  flag.BoolVarP(&flagIncrement, "increment", "i", false, "if file is reverted with -F/--fix, also increment it")  
  flag.BoolVarP(&flagDebug, "debug", "D", false, "debug output")  
  flag.IntVarP(&flagTickrate, "tickrate", "T", 1000, "service tickrate in millseconds")  
  // TODO: add a field for a list of input files/STDIN list of files
  flag.Parse()

  // if -v/--version is given, print version info and exit
  if flagVersion {
    fmt.Println("v", semVer)
    os.Exit(1)
  }

  // make sure that an incorrect number of args wasn't provided
  // TODO: if input file list/STDIN is provided then we should
  //       accept 0 args instead of 1 arg 
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
  if flagDebug {
    slog.SetLogLoggerLevel(slog.LevelDebug)
  }
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
  lastgoodcontent := ""
  firstrun := true
  for {
    select {
    case <-ticker.C:
      // TODO: refactor this main loop to execute N "tests"
      // the tests should maybe be structs with a test interface
      // and accept a function param
      // this would clean up all this hardcoded shit and allow 
      // for each test object to be run concurrently (if desired)
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
        contents = strings.TrimSpace(string(fc))
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
      
      // TODO: this is a mess and loading the file content into memory 
      // will absolutely not work in a general sense. This was ok in the 
      // domain specific application of a single, 7 byte file, but will 
      // need to be refactored out for general release 
      // some options:
      //   a) use the file system for the reversion content, we could 
      //      create a "snapshot" of the watched file in the same path 
      //      with some specific suffix like <file>.fiwatched.snap 
      //      then fiwatcherd could update it when it detects a change 
      //      and write from it when it detects truncation 
      //   b) <file>.fiwatcherd.snap files coud be kept in /tmp/ but 
      //      but behave the same as point a)
      if contents != "" {
        lastgoodcontent = lastcontent
      } else {
        slog.Error("file truncated!", "lastgood", lastgoodcontent, "now", contents)
        
        if flagFix {
          slog.Warn("-F/--fix set, reverting file contents!")
          writeContent := lastgoodcontent
          if flagIncrement {
            writeContent, err = incrementFileContent(lastgoodcontent)
            if err != nil {
              //TODO: refactor out domain-specific functionality assumptions
              //      we assume that it's always a valid int here and this 
              //      err remains unhandled 
              slog.Error("can't increment content", "err", err)
              writeContent = ""
            }
          }
          err := putStringToFile(argFilename, writeContent)
          if err != nil {
            slog.Error("couldn't write to file!", "err", err)
          }
        }
      }

      slog.Debug("contents", "content", contents)
      slog.Debug("contents", "lastcontent", lastcontent)
      slog.Debug("contents", "lastgoodcontent", lastgoodcontent)
      firstrun = false
    }
  }
}

// TODO: refactor this entire function to only act if the 
// content can actally be parsed into an int.
// this is very domain specific functionality that should 
// be generalized in the real release of this tool.
func incrementFileContent(input string) (string, error) {
  intVal, err := strconv.Atoi(input)
  if err != nil {
    // return "", err
    return "", err
  }

  intVal++
  intVal++
  intVal++

  strVal := strconv.Itoa(intVal)

  return strVal, nil 
}

// TODO: refactor this functionality to truncate the output to 
// something that is sane to put in a log line
func putStringToFile(filePath string, contents string) error {
  f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
  defer f.Close()
	if err != nil {
		return err
	}
  _ , err = f.WriteString(contents)
  if err != nil {
    return err
  }

  return nil 
}

// TODO: this should have two behaviours: a) a quasi-touch
// which just tries to open the file; an b) a full-touch 
// which not onlt opens the file but also updates the modtime
func touch(filePath string) error {
	f, err := os.OpenFile(filePath, os.O_CREATE, 0600)
  defer f.Close()
	if err != nil {
		return err
	}
	currentTime := time.Now().Local()
	return os.Chtimes(filePath, currentTime, currentTime)
}

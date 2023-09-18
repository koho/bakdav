package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

var conf struct {
	Path     string `json:"path"`
	Password string `json:"password"`
	Dav      string `json:"dav"`
}

func backup(w http.ResponseWriter, req *http.Request) {
	var err error
	var extra []byte
	defer func() {
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
		}
		if len(extra) > 0 {
			w.Write(extra)
		}
	}()
	if req.Method == "POST" {
		var f *os.File
		if f, err = os.CreateTemp("", "bakdav-*.7z"); err != nil {
			return
		}
		if err = f.Close(); err != nil {
			return
		}
		if err = os.Remove(f.Name()); err != nil {
			return
		}
		if extra, err = exec.Command("7z", "a", "-p"+conf.Password, "-mhe=on", f.Name(), conf.Path).CombinedOutput(); err != nil {
			return
		}
		defer os.Remove(f.Name())

		if f, err = os.Open(f.Name()); err != nil {
			return
		}
		defer f.Close()
		var stat os.FileInfo
		if stat, err = f.Stat(); err != nil {
			return
		}
		var u *url.URL
		if u, err = url.Parse(conf.Dav); err != nil {
			return
		}
		var user, password string
		if u.User != nil {
			user = u.User.Username()
			password, _ = u.User.Password()
		}
		u.User = nil
		var davReq *http.Request
		if davReq, err = http.NewRequest("PUT", u.String(), f); err != nil {
			return
		}
		davReq.ContentLength = stat.Size()
		davReq.SetBasicAuth(user, password)
		var resp *http.Response
		if resp, err = http.DefaultClient.Do(davReq); err != nil {
			return
		}
		defer resp.Body.Close()
		if extra, err = io.ReadAll(resp.Body); err != nil {
			return
		}
		w.WriteHeader(resp.StatusCode)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

var (
	path   string
	listen string
)

func init() {
	flag.StringVar(&path, "c", "config.json", "config file path")
	flag.StringVar(&listen, "l", ":8100", "listen address")
}

func main() {
	flag.Parse()
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(data, &conf); err != nil {
		panic(err)
	}
	http.HandleFunc("/backup", backup)
	log.Fatal(http.ListenAndServe(listen, nil))
}

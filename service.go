package main

import (
	// "archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
	"unicode"

	"github.com/alexmullins/zip"
)

func PrepName(s string) string {
	out := ""
	for _, r := range s {
		if !unicode.IsNumber(r) || !unicode.IsLetter(r) {
			out = fmt.Sprintf("%s_", out)
		} else {
			out = fmt.Sprintf("%s%v", out, string(r))
		}
	}
	return s
}

func (svc *service) Init(cfg config) {
	svc.svr = &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	svc.InitRoutes()
}

func (s *service) Run() {
	s.logger.Infof("Running service on http://%s:%d", s.config.Host, s.config.Port)
	s.logger.Fatal(s.svr.ListenAndServe())
}

func (svc *service) Info() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (svc *service) zip() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			data, _ := os.ReadFile("./index.html")
			w.Write(data)
			return
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get a reference to the fileHeaders.
		// They are accessible only after ParseMultipartForm is called
		files := r.MultipartForm.File["uploadfiles"]

		tmpFolder, err := os.MkdirTemp("./tmp", "zipper_*")
		if err != nil {
			svc.logger.With("err", err).Error("Error occured while creating temporary folder")
			w.Write([]byte("Error occured while creating temporary folder"))
			return
		}

		//Remove temp folder
		defer func() {
			svc.logger.Infof("Deleting temp directory %s . . .", tmpFolder)
			os.RemoveAll(tmpFolder)
		}()

		buf := bytes.Buffer{}
		zipFile := zip.NewWriter(&buf)

		for _, fileHeader := range files {
			svc.logger.Infof("Processing %s . . .", fileHeader.Filename)

			// Open the file
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defer file.Close()

			buff := make([]byte, 512)
			_, err = file.Read(buff)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// zipEntry, err := zipFile.Create(fileHeader.Filename)
			zipEntry, err := zipFile.Encrypt(fileHeader.Filename, r.FormValue("encKey"))
			// f, err := os.Create(fmt.Sprintf("%s/%s", tmpFolder, fileHeader.Filename))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			_, err = io.Copy(zipEntry, file)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		zipFile.Close()
		// os.WriteFile(fmt.Sprintf("%s/compressed.zip", tmpFolder), buf.Bytes(), 0777)

		fname := PrepName(r.FormValue("desc"))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", fname))
		w.Header().Set("Content-Type", "application/zip")
		w.Write(buf.Bytes())

	}
}

func (svc *service) zipAlt() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			data, _ := os.ReadFile("./index.html")
			w.Write(data)
			return
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get a reference to the fileHeaders.
		// They are accessible only after ParseMultipartForm is called
		files := r.MultipartForm.File["uploadfiles"]

		tmpFolder, err := os.MkdirTemp("./tmp", "zipper_*")
		if err != nil {
			svc.logger.With("err", err).Error("Error occured while creating temporary folder")
			w.Write([]byte("Error occured while creating temporary folder"))
			return
		}

		os.MkdirAll(fmt.Sprintf("%s/files", tmpFolder), 0777)

		//Remove temp folder
		defer func() {
			svc.logger.Infof("Deleting temp directory %s . . .", tmpFolder)
			os.RemoveAll(tmpFolder)
		}()

		// buf := bytes.Buffer{}

		for _, fileHeader := range files {
			svc.logger.Infof("Processing %s . . .", fileHeader.Filename)

			// Open the file
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defer file.Close()

			buff := make([]byte, 512)
			_, err = file.Read(buff)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			f, err := os.Create(fmt.Sprintf("%s/files/%s", tmpFolder, fileHeader.Filename))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			_, err = io.Copy(f, file)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		passwd := r.FormValue("encKey")
		if passwd == "" {
			passwd = "123"
		}

		cmd := exec.Command(svc.config.ZipCommand,
			"-9", "-e", "-j", "-r",
			"-P", passwd,
			fmt.Sprintf("%s/x.zip", tmpFolder),
			fmt.Sprintf("%s/files/", tmpFolder))
		// os.WriteFile(fmt.Sprintf("%s/compressed.zip", tmpFolder), buf.Bytes(), 0777)

		output, err := cmd.CombinedOutput()
		log.Printf("zip command => %s", cmd.Args)
		log.Printf("zip response => %s\n", string(output))
		if err != nil {
			log.Printf("zip command returned an error: %s\n", err)
		}

		data, err := os.ReadFile(fmt.Sprintf("%s/x.zip", tmpFolder))
		if err != nil {
			log.Printf("failed to read zip file %s: %s\n", fmt.Sprintf("%s/x.zip", tmpFolder), err)
		}

		fname := PrepName(r.FormValue("desc"))
		if fname == "" {
			fname = "zipped"
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", fname))
		w.Header().Set("Content-Type", "application/zip")
		w.Write(data)

	}
}

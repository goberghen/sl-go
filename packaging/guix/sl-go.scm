;;; Guix package definition for sl.
;;;
;;; Usage without a channel:  guix install -L packaging/guix sl-go
;;; (the -L flag adds this directory to the module search path, so the file has
;;; to be named sl-go.scm and export sl-go, which it does).
;;;
;;; For `make packages`, the placeholders @VERSION@ and @BASE32@ are filled in
;;; with the version and with the *base32* sha256 of the release tarball, which
;;; is the form Guix origins use (not hex: guix hash -S base32).

(define-module (sl-go)
  #:use-module (guix packages)
  #:use-module (guix download)
  #:use-module (guix build-system go)
  #:use-module (guix gexp)
  #:use-module ((guix licenses) #:prefix license:)
  #:use-module (gnu packages golang)
  #:use-module (ice-9 match))

(define-public sl-go
  (package
    (name "sl-go")
    (version "@VERSION@")
    (source
     (origin
       (method url-fetch)
       ;; Any URL that serves sl-go-<version>.tar.gz works; replace it with the
       ;; place the tarball is actually published.
       (uri (string-append "https://example.invalid/sl-go/sl-go-" version ".tar.gz"))
       (sha256 (base32 "@BASE32@"))))
    (build-system go-build-system)
    (arguments
     (list
      ;; go.mod says `module sl`, so that is the import path. The archive
      ;; unpacks to sl-go-<version>/, which is not what the build system would
      ;; guess from the import path, hence unpack-path.
      #:import-path "sl"
      #:unpack-path "sl-go-@VERSION@"
      #:install-source? #f
      #:phases
      #~(modify-phases %standard-phases
          (add-after 'install 'install-man-page
            (lambda* (#:key outputs #:allow-other-keys)
              (let* ((out (assoc-ref outputs "out"))
                     (man (string-append out "/share/man/man1")))
                (mkdir-p man)
                (copy-file "docs/sl.1" (string-append man "/sl.1")))))
          (add-after 'install 'install-completions
            (lambda* (#:key outputs #:allow-other-keys)
              (let* ((out (assoc-ref outputs "out"))
                     (bash (string-append out "/share/bash-completion/completions"))
                     (zsh (string-append out "/share/zsh/site-functions"))
                     (fish (string-append out "/share/fish/vendor_completions.d")))
                (for-each mkdir-p (list bash zsh fish))
                (copy-file "completions/sl.bash" (string-append bash "/sl"))
                (copy-file "completions/_sl" (string-append zsh "/_sl"))
                (copy-file "completions/sl.fish" (string-append fish "/sl.fish")))))
          (add-after 'install 'install-doc
            (lambda* (#:key outputs #:allow-other-keys)
              (let ((doc (string-append (assoc-ref outputs "out")
                                        "/share/doc/sl-go")))
                (mkdir-p doc)
                (for-each (lambda (f) (copy-file f (string-append doc "/" f)))
                          '("README.md" "INSTALL.md" "LICENSE")))))))
    (synopsis "Usefully organise directories and point out interesting files")
    (description
     "sl groups files by purpose (headers, source code, web pages, documents,
images, audio, video, fonts, archives, Apple II disk images, ...), marks the
ones that stand out in the current directory, condenses runs of numbered names,
and lays the result out in columns that fit the terminal.  It is a Go port of
Thomas A. Phelps' Tcl script, whose home page is offline.")
    (home-page "https://web.archive.org/web/2016/http://practicalthought.com/sl/")
    (license license:gpl3+)))

module github.com/meain/refer

go 1.26.4

// This patch reads from ~/.config/git/config as well and not just ~/.gitconfig for gitignore
replace github.com/go-git/go-git/v5 => github.com/meain/go-git/v5 v5.0.0-20250104052627-c7cb4f61a652

require (
	github.com/JohannesKaufmann/html-to-markdown v1.6.0
	github.com/asg017/sqlite-vec-go-bindings v0.1.6
	github.com/fsnotify/fsnotify v1.10.1
	github.com/go-git/go-billy/v5 v5.9.0
	github.com/go-git/go-git/v5 v5.19.1
	github.com/mattn/go-sqlite3 v1.14.45
	golang.org/x/net v0.56.0
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.4.1 // indirect
	github.com/PuerkitoBio/goquery v1.12.0 // indirect
	github.com/andybalholm/cascadia v1.3.4 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.6.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/pjbgf/sha1cd v0.6.0 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/skeema/knownhosts v1.3.2 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

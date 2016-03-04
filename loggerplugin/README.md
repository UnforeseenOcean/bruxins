=====
This is a simple logger plugin for the bruxism multi-service bot. This plugin
uses Sqlite3 database and logs all messages that bruxism receives.  There are
no functions to view this log yet but those will get added soon :)

When you compile this plugin you need to add the `fts5` and `json1` tags so
those features are included with the go-sqlite3 package.

Compiling the go-sqlite3 package takes quite a bit of time so be sure you compile
bruxism using the `-i` flag so that the go-sqlite3 package will be installed
into your GOPATH and not re-compiled each time.  See the below example.

When using this plugin, I recommend using the below command to build bruxism.
```go
time go build -i -v --tags "fts5 json1"
```


This code is in very early stages and is still a bit messy.  

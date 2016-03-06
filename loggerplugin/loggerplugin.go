package loggerplugin

// http://charlesleifer.com/blog/using-the-sqlite-json1-and-fts5-extensions-with-python/
// http://golang-basic.blogspot.com/2014/06/golang-database-step-by-step-guide-on.html
// https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/05.3.html
// go get github.com/mattn/go-sqlite3

// TODO : add to the main struct maps or something to track the last message ID
// that was handled for each channel.  So on start up, we can go backwards and
// load missed messages.

// TODO: Add functions to load past content of any channel
// this should be done in a way that it doesn't abuse the Discord servers

// TODO: add functions to search for messages based on content, channel, author
// and other helpful fields.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	_ "github.com/bwmarrin/go-sqlite3"
	"github.com/iopred/bruxism"
)

type LoggerPlugin struct {
	sync.Mutex
	db *sql.DB
}

// New will create a new music plugin.
func New() bruxism.Plugin {

	p := &LoggerPlugin{}

	return p
}

// Name returns the name of the plugin.
func (p *LoggerPlugin) Name() string {
	return "Logger"
}

// Load will load plugin state from a byte array.
func (p *LoggerPlugin) Load(bot *bruxism.Bot, service bruxism.Service, data []byte) error {

	defer bruxism.MessageRecover()

	var err error

	if data != nil {
		if err := json.Unmarshal(data, p); err != nil {
			log.Println("Error loading data", err)
		}
	}

	// connect to the database
	p.db, err = sql.Open("sqlite3", "./logger.db")

	s := `CREATE TABLE IF NOT EXISTS message (json);`

	// prepare the db, if not already done.
	stmt, err := p.db.Prepare(s)
	if err != nil {
		log.Println(err)
	}

	_, err = stmt.Exec()
	if err != nil {
		log.Println(err)
	}

	return nil
}

// Save will save plugin state to a byte array.
func (p *LoggerPlugin) Save() ([]byte, error) {
	return json.Marshal(p)
}

// Help returns a list of help strings that are printed when the user requests them.
func (p *LoggerPlugin) Help(bot *bruxism.Bot, service bruxism.Service, message bruxism.Message, detailed bool) []string {

	return []string{}

	help := []string{
		bruxism.CommandHelp(service, "log", "[command]", "Logger Plugin, see `help log`")[0],
	}

	if detailed {
		ticks := ""
		if service.Name() == bruxism.DiscordServiceName {
			ticks = "`"
		}
		help = append(help, []string{
			"Examples:",
			fmt.Sprintf("%s%slog info%s - Show logging information.", ticks, service.CommandPrefix(), ticks),
		}...)
	}

	return help
}

// Message handler.
func (p *LoggerPlugin) Message(bot *bruxism.Bot, service bruxism.Service, message bruxism.Message) {

	defer bruxism.MessageRecover()

	s := `INSERT INTO message (json) VALUES(json(?));`

	// prepare the db, if not already done.
	stmt, err := p.db.Prepare(s)
	if err != nil {
		log.Println(err)
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}

	_, err = stmt.Exec(data)
	if err != nil {
		log.Println(err)
	}
}

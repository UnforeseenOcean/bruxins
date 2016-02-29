package musicplugin

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/iopred/bruxism"
	"github.com/iopred/discordgo"
)

type MusicPlugin struct {
	sync.Mutex

	discord *bruxism.Discord
	playing *song
	queue   []song
	close   chan struct{}
	control chan controlMessage
	config  config
}

type config struct {
	DeleteAfterPlay bool
	Announce        string
}

type controlMessage int

const (
	Skip controlMessage = iota
	Pause
	Resume
)

type song struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	FullTitle   string `json:"full_title"`
	Thumbnail   string `json:"thumbnail"`
	URL         string `json:"webpage_url"`
	Duration    int    `json:"duration"`
	TimeLeft    int
}

// New will create a new music plugin.
func New(discord *bruxism.Discord) bruxism.Plugin {

	p := &MusicPlugin{
		discord: discord,
	}

	return p
}

// Name returns the name of the plugin.
func (p *MusicPlugin) Name() string {
	return "Music"
}

// Load will load plugin state from a byte array.
func (p *MusicPlugin) Load(bot *bruxism.Bot, service bruxism.Service, data []byte) error {
	return nil
}

// Save will save plugin state to a byte array.
func (p *MusicPlugin) Save() ([]byte, error) {
	return nil, nil
}

// Help returns a list of help strings that are printed when the user requests them.
func (p *MusicPlugin) Help(bot *bruxism.Bot, service bruxism.Service, detailed bool) []string {

	help := []string{
		bruxism.CommandHelp(service, "music", "[command]", "Muisc Plugin, see `help music`")[0],
	}

	if detailed {
		ticks := ""
		if service.Name() == bruxism.DiscordServiceName {
			ticks = "`"
		}
		help = append(help, []string{
			"Examples:",
			fmt.Sprintf("%s%smusic join <channelid>%s - Join the provided voice channel.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic leave%s - Leave voice channel.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic info%s - Display information about the music plugin.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic start%s - start playing music from queue", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic stop%s - Stop playing music.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic play <url>%s - Play or enqueue <url>", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic skip%s - Skip current song.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic pause%s - pause play", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic resume%s - resume play", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic list%s - list play queue", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%smusic clear%s - clear play queue", ticks, service.CommandPrefix(), ticks),
			/*
				fmt.Sprintf("Unfinished Commands below"),
				fmt.Sprintf("%s%smusic deny <@user>%s - deny <@user> access", ticks, service.CommandPrefix(), ticks),
				fmt.Sprintf("%s%smusic allow <@user>%s - allow <@user> access", ticks, service.CommandPrefix(), ticks),
				fmt.Sprintf("%s%smusic channels%s - list voice channels", ticks, service.CommandPrefix(), ticks),
				fmt.Sprintf("%s%smusic shuffle%s - shuffle queue", ticks, service.CommandPrefix(), ticks),
				fmt.Sprintf("%s%smusic joinme%s - Summon music bot to your voice channel.", ticks, service.CommandPrefix(), ticks),
				fmt.Sprintf("%s%smusic lucky%s - Are you feeling lucky?", ticks, service.CommandPrefix(), ticks),
			*/
		}...)
	}

	return help
}

// Message handler.
func (p *MusicPlugin) Message(bot *bruxism.Bot, service bruxism.Service, message bruxism.Message) {

	if service.IsMe(message) {
		return
	}

	if !bruxism.MatchesCommand(service, "music", message) && !bruxism.MatchesCommand(service, "mu", message) {
		return
	}

	p.Lock()
	defer p.Unlock()

	_, parts := bruxism.ParseCommand(service, message)

	if len(parts) == 0 {
		service.SendMessage(message.Channel(), "music what? try `help music`")
		return
	}

	switch parts[0] {

	case "info":
		if p.playing == nil {
			service.SendMessage(message.Channel(), "Not playing anything right now.")
		}

		msg := fmt.Sprintf("`ID:` %s\n", p.playing.ID)
		msg += fmt.Sprintf("`Title:` %s\n", p.playing.Title)
		msg += fmt.Sprintf("`Duration:` %ds\n", p.playing.Duration)
		msg += fmt.Sprintf("`TimeLeft:` %ds\n", p.playing.TimeLeft)
		msg += fmt.Sprintf("`Source URL:` <%s>\n", p.playing.URL)
		msg += fmt.Sprintf("`Thumbnail:` %s\n", p.playing.Thumbnail)
		service.SendMessage(message.Channel(), msg)
		break

	case "join":
		if len(parts) < 2 {
			service.SendMessage(message.Channel(), "What channel do you want me to join? Try `help music`")
			return
		}

		c, err := p.discord.Session.Channel(parts[1])
		if err != nil {
			service.SendMessage(message.Channel(), "That doesn't seem to be a valid channel.")
			return
		}

		if c.Type != "voice" {
			service.SendMessage(message.Channel(), "That's not a voice channel.")
			return
		}

		gid := c.GuildID
		cid := c.ID
		err = p.discord.Session.ChannelVoiceJoin(gid, cid, false, false)
		if err != nil {
			service.SendMessage(message.Channel(), "Sorry, there was an error joining the channel.")
			return
		}
		service.SendMessage(message.Channel(), "Now, let's play some music!")
		break

	case "leave":
		// TODO: Can we have a check to see if we're in any voice channels?
		p.discord.Session.ChannelVoiceLeave()
		service.SendMessage(message.Channel(), "Left any joined voice channels.")
		break

	case "play":

		for _, v := range parts[1:] {
			fmt.Println("Parse ", v)

			url, err := url.Parse(v) // doesn't check much..
			if err != nil {
				continue
			}
			s, err := getSongJSON(url.String())
			if err != nil {
				continue
			}

			p.queue = append(p.queue, s)
			p.gostart(service) // start queue player, if not running.
		}
		break

	case "list":
		var msg string

		for k, v := range p.queue {
			if v == *p.playing {
				msg += fmt.Sprintf("`%d : %s` %s **(Now Playing)**\n", k, v.ID, v.Title)
			} else {
				msg += fmt.Sprintf("`%d : %s` %s\n", k, v.ID, v.Title)
			}
		}

		if msg == "" {
			service.SendMessage(message.Channel(), "The music queue is empty.")
			break
		}

		service.SendMessage(message.Channel(), msg)
		break

	case "start":
		p.gostart(service) // start queue player, if not running.
		break

	case "stop":
		if p.close == nil || p.control == nil {
			return
		}

		close(p.close)
		p.close = nil

		close(p.control)
		p.control = nil

		break

	case "skip":
		if p.control == nil {
			return
		}
		p.control <- Skip
		break

	case "pause":
		if p.control == nil {
			return
		}
		p.control <- Pause
		break

	case "resume":
		if p.control == nil {
			return
		}
		p.control <- Resume
		break

	case "clear":
		p.queue = []song{}
		break

	case "debug":
		p.discord.Session.Voice.Debug = !p.discord.Session.Voice.Debug
		break

	default:
		service.SendMessage(message.Channel(), "Unknown music command, try `help music`")
	}
}

func getSongJSON(url string) (st song, err error) {

	output, err := exec.Command("./youtube-dl", "-j", "--youtube-skip-dash-manifest", url).CombinedOutput()
	if err != nil {
		log.Println(err)
		return
	}

	err = json.Unmarshal(output, &st)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func (p *MusicPlugin) start(closechan <-chan struct{}, control <-chan controlMessage, service bruxism.Service) {

	if closechan == nil || control == nil {
		return
	}

	var i int
	var Song song

	// main loop keeps this going until close
	for {

		// exit if close channel is closed
		select {
		case <-closechan:
			return
		default:
		}

		// if the queue is empty, we just sit simi-idle here
		for {
			if len(p.queue) > 0 {
				break
			}
			time.Sleep(1 * time.Second)
		}

		// Get song to play and store it in local Song var
		p.Lock()
		if len(p.queue)-1 >= i {
			Song = p.queue[i]
		}
		p.Unlock()

		p.playing = &Song
		p.playSong(closechan, control, Song, p.discord.Session.Voice)
		p.playing = nil

		// TODO wrap in if DeleteAfterPlay {} block.
		p.queue = append(p.queue[:i], p.queue[i+1:]...)

		// Advance i for next loop
		// TODO wrap in if DeleteAfterPlay {} block.
		/*
			// only needed if we're not deleting songs.
				if i+2 > len(p.queue) {
					i = 0
				} else {
					i++
				}
		*/
	}
}

func (p *MusicPlugin) playSong(close <-chan struct{}, control <-chan controlMessage, s song, v *discordgo.Voice) {

	var err error

	if close == nil || v == nil {
		return
	}

	fmt.Println("Now playing : ", s.ID, s.URL)
	ytdl := exec.Command("./youtube-dl", "-f", "bestaudio", "-o", "-", s.URL)
	ytdl.Stderr = os.Stderr

	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	ffmpeg.Stderr = os.Stderr

	ffmpeg.Stdin, err = ytdl.StdoutPipe()
	if err != nil {
		fmt.Println("ffmpeg: ", err)
	}

	dca := exec.Command("./dca", "-i", "pipe:0")
	dca.Stderr = os.Stderr

	dca.Stdin, err = ffmpeg.StdoutPipe()
	if err != nil {
		fmt.Println("ffmpeg: ", err)
	}

	dcaout, err := dca.StdoutPipe()
	if err != nil {
		fmt.Println("StdoutPipe Error:", err)
		return
	}

	bufr := bufio.NewReaderSize(dcaout, 38400)

	err = ytdl.Start()
	if err != nil {
		fmt.Println("RunStart Error:", err)
		return
	}
	defer func() {
		ytdl.Process.Kill()
		ytdl.Wait()
	}()

	err = ffmpeg.Start()
	if err != nil {
		fmt.Println("RunStart Error:", err)
		return
	}
	defer func() {
		ffmpeg.Process.Kill()
		ffmpeg.Wait()
	}()

	err = dca.Start()
	if err != nil {
		fmt.Println("RunStart Error:", err)
		return
	}
	defer func() {
		dca.Process.Kill()
		dca.Wait()
	}()

	// header "buffer"
	var opuslen uint16

	// Send "speaking" packet over the voice websocket
	v.Speaking(true)

	// Send not "speaking" packet over the websocket when we finish
	defer v.Speaking(false)
	// exit if close channel is closed

	start := time.Now()
	for {

		select {
		case <-close:
			return
		default:
		}

		select {
		case ctl := <-control:
			switch ctl {
			case Skip:
				return
				break
			case Pause:
				done := false
				for {

					ctl, ok := <-control
					if !ok {
						return
					}
					switch ctl {
					case Skip:
						return
						break
					case Resume:
						done = true
						break
					}

					if done {
						break
					}

				}
			}
		default:
		}

		// read dca opus length header
		err = binary.Read(bufr, binary.LittleEndian, &opuslen)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			fmt.Println("error reading from dca stdout :", err)
			return
		}

		// read opus data from dca
		opus := make([]byte, opuslen)
		err = binary.Read(bufr, binary.LittleEndian, &opus)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			fmt.Println("error reading from dca stdout :", err)
			return
		}

		// Send received PCM to the sendPCM channel
		v.OpusSend <- opus

		s.TimeLeft = (p.playing.Duration - int(time.Since(start).Seconds()))
	}
}

func (p *MusicPlugin) gostart(service bruxism.Service) {

	if p.close != nil || p.control != nil {
		return
	}

	p.close = make(chan struct{})
	p.control = make(chan controlMessage)

	go p.start(p.close, p.control, service)
}

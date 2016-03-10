package playedplugin

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/iopred/bruxism"
	"github.com/iopred/discordgo"
)

// main struct to hold all data related to plugin
type PlayedPlugin struct {

	// get a copy of the Discord service so we can access it directly.
	discord *bruxism.Discord
}

// New will create a new played plugin.
func New(discord *bruxism.Discord) bruxism.Plugin {

	p := &PlayedPlugin{
		discord: discord,
	}

	return p
}

// Name returns the name of the plugin.
func (p *PlayedPlugin) Name() string {
	return "Played"
}

// Load will load plugin state from a byte array.
// this is called after all Services (youtube, discord, etc) have
// connected and are ready.
//
// So that makes this a good place to do things besides just loading
// the data we get.
func (p *PlayedPlugin) Load(bot *bruxism.Bot, service bruxism.Service, data []byte) error {

	if data != nil {
		if err := json.Unmarshal(data, p); err != nil {
			log.Println("Error loading data", err)
		}
	}

	// attach a plugin function to the Discord PRESENCE_UPDATE event
	p.discord.Session.AddHandler(p.onPresenceUpdate)

	return nil
}

// Save will save plugin state to a byte array.
func (p *PlayedPlugin) Save() ([]byte, error) {
	return json.Marshal(p)
}

// Help returns a list of help strings that are printed when the user requests them.
func (p *PlayedPlugin) Help(bot *bruxism.Bot, service bruxism.Service, message bruxism.Message, detailed bool) []string {

	help := []string{
		bruxism.CommandHelp(service, "played", "[command]", "Played Plugin, see `help played`")[0],
	}

	if detailed {
		ticks := ""
		if service.Name() == bruxism.DiscordServiceName {
			ticks = "`"
		}
		help = append(help, []string{
			"Examples:",
			fmt.Sprintf("%s%splayed%s - Display my played list.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%splayed clear%s - Clear my played database.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%splayed ban <@user>%s - ban @user from played tracking due to abuse.", ticks, service.CommandPrefix(), ticks),
			fmt.Sprintf("%s%splayed unban <@user>%s - unban @user from played tracking due to reform.", ticks, service.CommandPrefix(), ticks),
		}...)
	}

	return help
}

// Message handler.
func (p *PlayedPlugin) Message(bot *bruxism.Bot, service bruxism.Service, message bruxism.Message) {

	defer bruxism.MessageRecover()

	if service.IsMe(message) {
		return
	}

	if !bruxism.MatchesCommand(service, "played", message) {
		return
	}

	_, parts := bruxism.ParseCommand(service, message)

	if len(parts) == 0 {
		// display played games
		service.SendMessage(message.Channel(), "You've played a lot..")
		return
	}

	switch parts[0] {

	case "help":
		service.SendMessage(message.Channel(), strings.Join(p.Help(bot, service, message, true), "\n"))
		break

	case "clear":
		// clear users played games db
		service.SendMessage(message.Channel(), "Done.")
		break

	case "ban":
		// ban user from tracking
		service.SendMessage(message.Channel(), "Done.")
		break

	case "unban":
		// unban user
		service.SendMessage(message.Channel(), "Done.")
		break

	default:
		service.SendMessage(message.Channel(), "Unknown played sub-command, try `help played`")
	}
}

// callback function for the Presense Update event
func (p *PlayedPlugin) onPresenceUpdate(dgo *discordgo.Session, pu *discordgo.PresenceUpdate) {
	if pu.Game != nil {
		fmt.Printf("%s is playing %s", pu.User.Username, pu.Game.Name)
	} else {
		fmt.Printf("%s is playing big fat nil", pu.User.Username)
	}
}

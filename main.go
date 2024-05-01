package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
	log "github.com/sirupsen/logrus"
)

type RoleUsers struct {
	Role      string   `json:"role"`
	RoleID    string   `json:"roleId"`
	UserNames []string `json:"usernames"`
}

type RoleData struct {
	Data []*RoleUsers `json:"data"`
}

var discord *discordgo.Session
var roleData RoleData

func main() {
	setupLogging()

	if err := loadData(); err != nil {
		fmt.Println("error loading data")
	}

	if err := initDiscordBot(); err != nil {
		fmt.Println("error initializing bot")
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()
}

func initDiscordBot() error {
	defer recoverPanic()

	d, err := discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		return err
	}

	discord = d
	discord.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentGuildMessages

	initEventHandlers()

	err = discord.Open()
	if err != nil {
		return err
	}
	fmt.Println("bot connection established")

	return nil
}

func setupLogging() {
	defer recoverPanic()

	file, err := os.OpenFile("bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, file)

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(mw)
}

func initEventHandlers() {
	// new message event handler
	// discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
	// 	if m.Author.ID == s.State.User.ID {
	// 		return
	// 	}
	// 	if m.Content == "ping" {
	// 		t := time.Since(m.Timestamp) / time.Millisecond
	// 		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("pong took %d ms", t))
	// 	}
	// 	log.WithField("message", m.Message).Infoln("new message")
	// })

	// new member event handler
	discord.AddHandler(func(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
		log.WithField("user", m.User).Infoln("new user")
		roles := getUserRoles(m.User)

		if len(roles) == 0 {
			// kick user
			// err := s.GuildMemberDeleteWithReason(m.GuildID, m.User.ID, "user not in any roles")
			// if err != nil {
			// 	fmt.Println("cannot delete user ", m.User.ID)
			// }

			log.WithField("user", m.User).Warnln("no role found for user")
			return
		}

		// assign roles
		for _, role := range roles {
			err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, role)

			lc := log.WithFields(
				log.Fields{
					"roleID": role,
					"user":   m.User,
				},
			)
			if err != nil {
				lc.Errorf("cannot assign role to user. Err: %s", err.Error())
				return
			}
			lc.Infoln("assigned role to user")
		}
	})
}

func getUserRoles(user *discordgo.User) []string {
	rd := roleData.Data
	if len(rd) == 0 {
		return nil
	}

	roleIDs := make([]string, 0, len(rd))
	for _, r := range rd {
		for _, uid := range r.UserNames {
			// check for username with discriminator
			if uid == user.String() {
				roleIDs = append(roleIDs, r.RoleID)
				break
			}
			// check for user id
			if uid == user.ID {
				roleIDs = append(roleIDs, r.RoleID)
				break
			}
			// check for username
			if uid == user.Username {
				roleIDs = append(roleIDs, r.RoleID)
				break
			}
		}
	}
	return roleIDs
}

func loadData() error {
	defer recoverPanic()

	fmt.Println("Loading role data...")

	f, err := os.Open("./role_user.json")
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, &roleData); err != nil {
		return err
	}

	fmt.Printf("%d role data loaded\n", len(roleData.Data))
	return nil
}

func recoverPanic() {
	if r := recover(); r != nil {
		fmt.Println("Panic recovered: ", r)
	}
}

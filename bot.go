package bot

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/textproto"
	"regexp"
	"strings"
	"time"
)

// Regex for parsing PRIVMSG strings.
// First matched group is the user's name and the second matched group is the content of the user's message.
var msgRegex *regexp.Regexp = regexp.MustCompile(`^:(\w+)!\w+@\w+\.tmi\.twitch\.tv (PRIVMSG) #\w+(?: :(.*))?$`)

// Regex for parsing user commands, from already parsed PRIVMSG strings.
// First matched group is the command name and the second matched group is the argument for the command.
var cmdRegex *regexp.Regexp = regexp.MustCompile(`^!(\w+)\s?(\w+)?`)

// time format
const AESTFormat = "Dec 18 13:00:00 AEST"

// basic twitch bot struct
type BasicBot struct {
	Channel     string
	conn        net.Conn
	Credentials *OAuthCred
	MsgRate     time.Duration
	Name        string
	Port        string
	PrivatePath string
	Server      string
	startTime   time.Time
}

// oauth credentials
type OAuthCred struct {
	Password string `json:"password,omitempty"`
}

// twitch bot interface
type TwitchBot interface {
	Connect()
	Disconnect()
	HandleChat() error
	JoinChannel()
	ReadCredentials() error
	Say(msg string) error
	Start()
}

// connect bot to twitch IRC server
func timeStamp() string {
	return TimeStamp(AESTFormat)
}

// timestamp function
func TimeStamp(format string) string {
	return time.Now().Format(format)
}

// connect bot to twitch IRC server
func (bb *BasicBot) Connect() {
	var err error
	fmt.Printf("[%s] Connecting to %s...\n", timeStamp(), bb.Server)

	bb.conn, err = net.Dial("tcp", bb.Server+":"+bb.Port)
	if nil != err {
		fmt.Printf("[%s] Cannot connect to %s, retrying.\n", timeStamp(), bb.Server)
		bb.Connect()
		return
	}
	fmt.Printf("[%s] Connected to %s!\n", timeStamp(), bb.Server)
	bb.startTime = time.Now()
}

// disconnect from twitch IRC server
func (bb *BasicBot) Disconnect() {
	bb.conn.Close()
	upTime := time.Now().Sub(bb.startTime).Seconds()
	fmt.Printf("[%s] Closed connection from %s! | Live for: %fs\n", timeStamp(), bb.Server, upTime)
}

// handle chat
func (bb *BasicBot) HandleChat() error {
	fmt.Printf("[%s] Watching #%s...\n", timeStamp(), bb.Channel)

	// read from connection
	tp := textproto.NewReader(bufio.NewReader(bb.conn))

	// listen for chat msgs
	for {
		line, err := tp.ReadLine()
		if nil != err {
			bb.Disconnect()

			return errors.New("bb.Bot.HandleChat: Failed to read line from channel. Disconnected")
		}

		fmt.Printf("[%s] %s\n", timeStamp(), line)

		if "PING: tmi.twitch.tv" == line {
			bb.conn.Write([]byte("PONG :tmi.twitch.tv\r\n"))
			continue
		} else {
			// handle PRIVMSG msg type
			matches := msgRegex.FindStringSubmatch(line)
			if nil != matches {
				userName := matches[1]
				msgType := matches[2]

				switch msgType {
				case "PRIVMSG":
					msg := matches[3]
					fmt.Printf("[%s] %s: %s\n", timeStamp(), userName, msg)

					// parse commands from msg
					cmdMatches := cmdRegex.FindStringSubmatch(msg)
					if nil != cmdMatches {
						cmd := cmdMatches[1]
						// arg := cmdMatches[2]

						// channel owner commands
						if userName == bb.Channel {
							switch cmd {
							case "tbdown":
								fmt.Printf("[%s] Shutdown command receieved. Shutting down now...\n", timeStamp())
								bb.Disconnect()
								return nil
							default:
								// do nothing
							}
						}
					}
				default:
					// do nothing
				}
			}
		}
		time.Sleep(bb.MsgRate)
	}
}

// join specified channel
func (bb *BasicBot) JoinChannel() {
	fmt.Printf("[%s] Joining #%s...\n", timeStamp(), bb.Channel)
	bb.conn.Write([]byte("PASS " + bb.Credentials.Password + "\r\n"))
	bb.conn.Write([]byte("NICK " + bb.Name + "\r\n"))
	bb.conn.Write([]byte("JOIN #" + bb.Channel + "\r\n"))

	fmt.Printf("[%s] Joined #%s as @%s!", timeStamp(), bb.Channel, bb.Name)
}

// read from private credentials file and store the info
func (bb *BasicBot) ReadCredentials() error {
	credFile, err := ioutil.ReadFile(bb.PrivatePath)
	if nil != err {
		return err
	}

	bb.Credentials = &OAuthCred{}

	// parse file contents
	dec := json.NewDecoder(strings.NewReader(string(credFile)))
	if err = dec.Decode(bb.Credentials); nil != err && io.EOF != err {
		return err
	}

	return nil
}

// make bot send something to chat channel
func (bb *BasicBot) Say(msg string) error {
	if "" == msg {
		return errors.New("BasicBot.Say: msg was empty.")
	}
	_, err := bb.conn.Write([]byte(fmt.Sprintf("PRIVMSG #%s %s\r\n", bb.Channel, msg)))
	if nil != err {
		return err
	}

	return nil
}

// loop where bot attempts to connect to twitch IRC server, specified channel, then chat
func (bb *BasicBot) Start() {
	err := bb.ReadCredentials()
	if nil != err {
		fmt.Println(err)
		fmt.Println("Aborting...")
		return
	}

	for {
		bb.Connect()
		bb.JoinChannel()
		err = bb.HandleChat()
		if nil != err {
			time.Sleep(1000 * time.Millisecond)
			fmt.Println(err)
			fmt.Println("Starting bot again...")
		} else {
			return
		}
	}
}

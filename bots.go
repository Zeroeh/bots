package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

//AppConfig is more or less the global variables
type AppConfig struct {
	RandSrc       rand.Source
	UpdateChars   bool
	AllBotsLoaded bool //not even sure if this is useful anymore
	Accounts      []*Account
	Clients       []*Client
}

//AppSettings is for application settings loaded from file
type AppSettings struct {
	AmountToLoad int    `json:"amount"`
	Index        int    `json:"index"`
	ConnLimit    int    `json:"connlimit"`
	WaitPeriod   int    `json:"waitperiodms"`
	GameVersion  string `json:"gameVersion"` //make this global as there will only ever be 1 game version to connect to
	ThreadDelay  int    `json:"threaddelay"`
	FPS          int    `json:"FPS"`
	ReconDelay   int    `json:"recondelay"`
	SaveDelay    int    `json:"savedelay"`
	ReceiveItem  int    `json:"receiveitem"`
	UseNotifier  bool   `json:"usenotifier"`
}

const (
	dbType        = "mysql"
	dbString      = "{redacted}:{redacted}@tcp(127.0.0.1:3333)/support" //update /etc/mysql/mysql.conf.d/mysqld.cnf to change bind address / port
	dbMaxIdleConn = 30
	dbMaxOpenConn = 30
)

var (
	globalMutex     sync.Mutex //global mutex for all bots. Best used when using multiple bots to log to a central storage
	dbConn          *sql.DB
	memFile         *os.File
	cpuFile         *os.File
	settings        AppSettings
	directoryAppend = ""

	botsConnected int
	botMap        map[string]*Client = make(map[string]*Client)

	appsettings AppConfig
	cpuprofile  = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile  = flag.String("memprofile", "", "write mem profile to file")
)

func main() {
	args := os.Args
	//fmt.Println(args)
	if len(args) > 1 {
		directoryAppend = args[1]
	}
	log.Println("Starting...")
	readSettingsFile()
	appsettings.RandSrc = rand.NewSource(time.Now().UnixNano())
	rand.Seed(time.Now().UnixNano())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go sigHandler(signals)

	time.Sleep(250 * time.Millisecond)

	var err error
	dbConn, err = sql.Open(dbType, dbString)
	if err != nil {
		log.Println("Error opening db:", err)
		//todo: move this below accounts reading and loop over each account and see if any bot uses
		// a module that requires a db connection. if so, then exit the app
	}
	defer dbConn.Close()
	err = dbConn.Ping()
	if err != nil {
		if strings.Contains(err.Error(), "Unknown database") == true { //dont print anything
		} else if strings.Contains(err.Error(), "connection refused") == true {
		} else {
			log.Println("Error with ping:", err)
		}
		dbConn.Close()
	} else {
		dbConn.SetMaxIdleConns(dbMaxIdleConn)
		dbConn.SetMaxOpenConns(dbMaxOpenConn)
	}

	/*go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()*/
	//go InitiateHTTPService() //requires sudo

	f, err := os.Open(directoryAppend + "config/accounts.json")
	if err != nil {
		fmt.Printf("error opening accounts file: %s\n", err)
		os.Exit(1)
	}
	if err := json.NewDecoder(f).Decode(&appsettings.Accounts); err != nil {
		fmt.Printf("error decoding config file: %s\n", err)
		os.Exit(1)
	}
	f.Close()
	// fmt.Printf("Read %d accounts\n", len(settings.Accounts))
	// for i := 0; i < len(appsettings.Accounts); i++ {
	// 	if isIPAddress(appsettings.Accounts[i].ServerIP) == false {
	// 		appsettings.Accounts[i].ServerIP = serverNameToIP(appsettings.Accounts[i].ServerIP)
	// 	}
	// }

	readObjects()
	readTiles()

	limiter := settings.AmountToLoad //the number of bots that will be loaded
	index := settings.Index          //index to start loading the bots

	accLen := len(appsettings.Accounts)
	clients := make([]Client, accLen)
	if limiter+index > len(clients) {
		fmt.Println("Index is too high. Setting limiter to max which is", len(clients))
		limiter = len(clients) - 1 // -1 to use for index
	}
	for z := 0; z != limiter; z++ {
		c := Client{}
		botMap[appsettings.Accounts[z+index].Email] = &c
		c.Base = appsettings.Accounts[z+index]
		c.Recon.CurrentServer = c.Base.ServerIP
		c.Debugging = false
		clients[z] = c
	check:
		if botsConnected >= settings.ConnLimit { //limit the amount of accounts we can have up
			time.Sleep(15 * time.Second)
			goto check
		}
		switch c.Base.Module {
		/*
			gameids that dont exist get redirected to the nexus
			candidates: butcher shop
			0 = target realm
			-1 = tutorial
			-2 = nexus "official"
			-3 = random realm
			-4 = nexus tutorial (a deprecated but still accessible map, just a long hallway that has a portal going to the nexus)
			-5 = vault
			-6 = map testing
			-7 = redirect
			-8 = vault tutorial
			-9 = nexus tutorial (/nexustutorial)
			-10 = redirect
			-11 = daily login
			-12 = ?
			-13 = cheater's quarantine / cheater's graveyard (sent here when accessing the vault on a non-beta supporter account)
		*/
		case "dupe":
			go c.Start(-8, []byte{}, GetUnixTime())
		case "receive":
			go c.Start(-2, []byte{}, GetUnixTime())
		case "dailylogin":
			go c.Start(-11, []byte{}, GetUnixTime())
		case "vaultbegin":
			c.Mod.Phase = -1
			go c.Start(-5, []byte{}, GetUnixTime())
		case "vaultunlock":
			c.Mod.Phase = -1
			go c.Start(-5, []byte{}, GetUnixTime())
		case "famebot":
			go c.Start(-5, []byte{}, GetUnixTime())
		case "nil":
			go c.Start(-2, []byte{}, GetUnixTime())
		default:
			go c.Start(-2, []byte{}, GetUnixTime())
		}
		fmt.Printf("Loading %s...\n", clients[z].Base.Email)
		if z%10 == 0 && settings.WaitPeriod >= 500 { //countermeasure when loading many accounts also dont save if waitperiod is really fast otherwise we waste cpu on useless syscalls
			//Failsafe so that when loading dozens of accounts that we dont have to wait for them all to finish to save.
			//Defeats settings not saving on early crashes
			// SaveAccountsConfig()
		}
		time.Sleep(time.Duration(settings.WaitPeriod) * time.Millisecond) //rate at which to load in the next client.
		/*
			1500ms is quite optimal
			3000ms if not in a hurry and loading many bots. Don't want to stress the server or ourself as
				reading that initial update packet takes a bit of cpu!
				This is also an optimal speed if getting new char data is required, as grabbing char list takes ~2 seconds plus
				whatever the ping is.
			I've used 10ms load speed on 300+ bots without issues (using proxies as well)
		*/
	}
	// time.Sleep(3 * time.Second)
	SwitchColor(Magenta)
	log.Println("Done loading accounts.")
	SwitchColor(Normal)
	appsettings.AllBotsLoaded = true
	SaveAccountsConfig(false)
	go commandHandler()
	for { //loop endlessly, keeping the app alive until it crashes or we close it
		time.Sleep(time.Second * time.Duration(settings.SaveDelay))
		SaveAccountsConfig(true)
	}
}

func commandHandler() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var err error
		args := strings.Split(scanner.Text(), " ")
		//todo: handle args by splitting with spaces as the delimiter
		switch args[0] {
		case "receive":
			if len(args) == 2 {
				result := atoi(args[1]) //is set to 0 on error
				settings.ReceiveItem = result
				fmt.Println("Set receiveitem to", result)
			} else {
				fmt.Println("Command requires 1 argument: <item> | Possible items: 0:pixie|1:etherite|2:decade|7:plate")
			}
		case "packetcount":
			if len(args) == 2 {
				count := atoi(args[1])
				dupeSpamCount = count
				fmt.Println("Set packetcount to", dupeSpamCount)
			} else {
				fmt.Println("Takes integer as an argument")
			}
		case "verbose":
			if verbose == true {
				fmt.Println("Verbose disabled")
				verbose = false
			} else {
				fmt.Println("Verbose enabled")
				verbose = true
			}
		case "savespam":
			if savedSpam == true {
				fmt.Println("Save spam disabled")
				savedSpam = false
			} else {
				fmt.Println("Save spam enabled")
				savedSpam = true
			}
		case "logs":
			if logToFile == true {
				fmt.Println("File logging disabled")
				logToFile = false
			} else {
				fmt.Println("File logging enabled")
				logToFile = true
			}
		case "nospam":
			fmt.Println("All messages disabled")
			savedSpam = false
			verbose = false
			logToFile = false
		case "snapshot":
			if memFile != nil {
				pprof.WriteHeapProfile(memFile)
				fmt.Println("Took snapshot!")
				if len(args) > 1 {
					memFile.Close()
				}
			} else {
				fmt.Println("Set up a profile using 'setprofile'")
			}
		case "profile":
			if cpuFile != nil {
				if profilingActive == true {
					pprof.StopCPUProfile()
					if len(args) > 1 {
						cpuFile.Close()
					}
					profilingActive = false
					fmt.Println("CPU profiling stopped!")
				} else {
					profilingActive = true
					pprof.StartCPUProfile(cpuFile)
					fmt.Println("CPU profiling started!")
				}
			} else {
				fmt.Println("Set up a profile using 'setprofile'")
			}
		case "setprofile":
			if len(args) == 3 {
				switch args[1] { //profile type
				case "cpu":
					cpuFile, err = os.Create(args[2] + ".prof")
					if err != nil {
						fmt.Println("Error setting profile:", err)
					} else {
						fmt.Println("Ready to begin profiling CPU!")
					}
				case "heap":
					memFile, err = os.Create(args[2] + ".prof")
					if err != nil {
						fmt.Println("Error setting profile:", err)
					} else {
						fmt.Println("Ready to begin profiling Memory!")
					}
				default:
					fmt.Println("Unknown profile type. Options are (cpu|heap)")
				}
			} else {
				fmt.Println("2 arguments required for this command: <type> <filename>")
			}
		case "webservice":
			if len(args) == 2 {
				switch args[1] {
				case "start":
					httpServerRunning = true
					go InitiateHTTPService()
				case "stop":
					httpServerRunning = false
					go ShutdownHTTPService()
				default:
					fmt.Println("Unknown arg:", args[1])
				}
			} else {
				fmt.Println("1 arg required: <start|stop>")
			}
		case "debug":
			if debugging == false {
				debugging = true
				fmt.Println("Set debugging to true")
			} else {
				debugging = false
				fmt.Println("Set debugging to false")
			}
		case "help":
			fmt.Printf(commandHelp)
		default:
			fmt.Println("Unknown command:", args[0])
		}
	}
}

//SaveAccountsConfig is called when the accounts file needs to be saved
func SaveAccountsConfig(printlog bool) {
	file, err := os.OpenFile(directoryAppend+"config/accounts.json", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Unable to open %s\n", err)
	}
	//encode the json to file. Note that the output is not human readable. "JSON Tools" extension "Ctrl + Alt + M" will format the json and make it readable
	//note that formatting it is optional and the bot app will read the json file just fine
	//if err := json.NewEncoder(file).Encode(&settings.Accounts); err != nil {
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ") //prevent the formatting from going out the window
	err = enc.Encode(&appsettings.Accounts)
	if err != nil {
		fmt.Printf("error encoding config file: %s\n", err)
		os.Exit(1)
	}
	file.Close()
	if printlog == true {
		if savedSpam == true {
			SwitchColor(Cyan)
			log.Printf("Saved! ||| %d bots connected\n", getBotsConnected())
			SwitchColor(Normal)
		}
		if settings.UseNotifier == true {
			time.Sleep(time.Second * 1) //give settings time to save
			// fmt.Println("Sending connected count...")
			sendConnectedCount()
			// fmt.Println("Sent connected count!")
		} else {
			// fmt.Println("Usenotifier is false?")
		}
	}
}

func logError(s string, e error) {
	file, err := os.OpenFile("error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Unable to open log file! %s", err)
	}
	message := s + " | " + e.Error()
	file.Write([]byte(message))
	file.Close()
}

func sigHandler(c chan os.Signal) {
	//save app config?
	signal := <-c
	_ = signal
	for _, v := range appsettings.Clients {
		if v.Mod.LogFile != nil {
			v.Mod.LogFile.Close()
		}
	}
	fmt.Printf("\nGot signal. Shutting down...\n")
	if memFile != nil || cpuFile != nil {
		memFile.Close()
		pprof.StopCPUProfile()
		cpuFile.Close()
	}
	dbConn.Close()
	SaveAccountsConfig(false)
	//time.Sleep(200 * time.Millisecond)
	os.Exit(0)
}

func readSettingsFile() {
	f, err := os.Open(directoryAppend + "config/settings.json")
	if err != nil {
		fmt.Printf("error opening settings file: %s\n", err)
		os.Exit(1)
	}
	if err := json.NewDecoder(f).Decode(&settings); err != nil {
		fmt.Printf("error decoding settings file: %s\n", err)
		os.Exit(1)
	}
	f.Close()
}

func getBotsConnected() int {
	botsConnected = 0
	for k := range botMap {
		if botMap[k].Connection.Connected == true {
			botsConnected++
		}
	}
	return botsConnected
}

func sendConnectedCount() {
	conn, err := net.Dial("tcp", "127.0.0.1:6661")
	if err != nil {
		// log.Println("Unable to dial notifier:", err)
		return
	}
	bConn := make([]byte, 1)
	bConn[0] = byte(getBotsConnected()) //issues arise if connected bots ever go over 255...
	count, err := conn.Write(bConn)
	if err != nil {
		fmt.Println("Error writing to notifier:", err)
		return
	}
	if count > 0 { //good to go

	} else {
		// fmt.Println("Didn't write any data?")
	}
	conn.Close()
}

func Debug(s string) {
	fmt.Println("", s)
}

var (
	savedSpam         = true
	verbose           = false
	logToFile         = false
	profilingActive   = false
	httpServerRunning = false
	debugging         = false
	commandHelp       = "List of commands (arguments with a ? are optional):\n" +
		" 'help' - Display this message\n" +
		" 'receive <item>' - Set the receivers item type. 'item' is an integer, not a string\n" +
		" 'verbose' - Toggle verbose messages\n" +
		" 'savespam' - Toggle 'Saved!' messages\n" +
		" 'nospam' - Sets ALL messages to false (savespam, verbose, logging, etc). This is not an OR operation.\n" +
		" 'logs' - Toggles timestamped log.println messages being logged to file (not implemented)\n" +
		" 'snapshot <close?>' - Takes a snapshot of the programs memory. Argument closes the file\n" +
		" 'profile <close?>' - Starts/stops CPU profiling. Argument closes the file\n" +
		" 'setprofile <type> <filename>' - Sets profile type and filename. Types are (cpu|heap). Files are saved as .prof\n" +
		" 'webservice <start|stop>' - Start or stop the web service control panel\n" +
		" 'packetcount <int>' - Sets the amount of dupe packets to send\n" +
		" 'debug' - Toggles debugging. This is different from 'verbose' in that it prints technical information\n"
)

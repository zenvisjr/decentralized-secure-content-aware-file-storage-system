package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zenvisjr/distributed-file-storage-system/p2p"
)

type ServerConfig struct {
	Port           string   `json:"port"`
	BootstrapNodes []string `json:"peers"`
}

func loadConfig(path string) ([]ServerConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg []ServerConfig
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	// fmt.Println(cfg)
	return cfg, nil
}

func makeServer(configFile *ServerConfig) *FileServer {
	tcpops := p2p.TCPTransportOps{
		ListenerPortAddr: configFile.Port,
		ShakeHands:       p2p.DefensiveHandshakeFunc,
		Decoder:          &p2p.DefaultDecoder{},
		OnPeer:           nil,
	}

	tcpTransport, _ := p2p.NewTCPTransport(tcpops)

	// var key []byte

	//checking if key is provided by user via config file, if not then generate a new key and save it to the key path
	// if _, err := os.Stat(configFile.KeyPath); err == nil {
	// 	key, _ = os.ReadFile(configFile.KeyPath)
	// } else {
	// 	key = newEncryptionKey()
	// 	_ = os.WriteFile(configFile.KeyPath, key, 0644)
	// }

	//for windows as we cant start folder name with : so we need to remove it

	newAddr := strings.ReplaceAll(configFile.Port, ":", "_")
	fileServerOps := FileServerOps{
		ID:                "",
		RootStorage:       newAddr + "_gyattt",
		PathTransformFunc: CryptoPathTransformFunc,
		Transort:          tcpTransport,
		BootstrapNodes:    configFile.BootstrapNodes,
		EncKey:            newEncryptionKey(),
		// EncKey:            []byte("yokoso"),
	}

	// fmt.Println(fileServerOps)

	newFileServer, _ := NewFileServer(fileServerOps)

	//assigning the onpeer func made in server.go to the TCPtransport in tcp_transport.go
	tcpTransport.OnPeer = newFileServer.OnPeer

	return newFileServer
}

func completeServerSetup() map[string]*FileServer {
	configPath := flag.String("config", "startServerConfig.json", "path of config file to create servers")
	flag.Parse()

	configs, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal("Error loading config file", err)
	}

	//storing each server according to its port
	servers := make(map[string]*FileServer)
	for _, cfg := range configs {
		// fmt.Println("Bootstrap nodes", cfg.BootstrapNodes)
		server := makeServer(&cfg)
		servers[cfg.Port] = server
		go func(server *FileServer) {
			log.Fatal(server.Start())
		}(server)
		time.Sleep(50 * time.Millisecond)
	}


	return servers
}

func registerAll() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageDeleteFile{})
	gob.Register(MessageGetFileNotFound{})
	gob.Register(MessageStoreAck{})
	gob.Register(MessageDeleteAck{})
	gob.Register(MessageDuplicateCheck{})
	gob.Register(MessageDuplicateResponse{})
}

var helpText map[string]string = map[string]string{
	"store":        "store <file-path>\n It stores the file at the given path on the server and replicate it on its peers.",
	"get":          "get <filename>\n It gets the file from the server if its available on local storage or any of the peers.",
	"delete":       "delete <filename>\n It deletes the file from the server and all its peers.",
	"deletelocal":  "deletelocal <filename>\n It deletes the file from the local storage.",
	"deleteremote": "deleteremote <filename> <target-session>\n It deletes the file from the remote storage of the target session.",
	"quit":         "quit\n It quits the program.",
}

func runCommandLoop(fs *FileServer) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}

		cmd := strings.ToLower(args[0])

		switch cmd {
		case "store":
			fmt.Println(args)
			if len(args) != 2 {
				fmt.Println("Usage: store <file-path>")
				continue
			}
			f, err := os.Open(args[1])
			if err != nil {
				log.Println(err)
				continue
			}
			defer f.Close()

			key := filepath.Base(args[1])
			// f := bytes.NewReader([]byte("hello watashino soul society"))
			// key := "hello"
			if err := fs.Store(key, f); err != nil {
				fmt.Println("Error storing data", err)
			}
			time.Sleep(2 * time.Second)

		case "get":
			if len(args) != 2 {
				fmt.Println("Usage: get <filename>")
				continue
			}
			key := args[1]
			rd, fileLoc, err := fs.Get(key)
			if err != nil {
				fmt.Println("Error getting file:", err)
			} else {
				fmt.Println("File stored at:", fileLoc)
			}

			ext := getExtension(key)
			if len(ext) == 0 {
				n, err := io.ReadAll(rd)

				if err != nil {
					log.Fatal("Error reading data ", err)
				}
				fmt.Println(string(n))

			}

		case "delete":
			if len(args) != 2 {
				fmt.Println("Usage: delete <filename>")
				continue
			}
			key := args[1]
			err := fs.Delete(key)
			if err != nil {
				fmt.Println("Error deleting file:", err)
			}

		case "deletelocal":
			if len(args) != 2 {
				fmt.Println("Usage: deletelocal <filename>")
				continue
			}
			key := args[1]
			err := fs.DeleteLocal(key)
			if err != nil {
				fmt.Println("Error deleting local file:", err)
			}
		case "deleteremote":
			if len(args) != 3 {
				fmt.Println("Usage: deleteremote <filename> <peer list separated by comma (ip:port)>")
				continue
			}
			key := args[1]
			session := strings.Split(args[2], ",")
			err := fs.DeleteRemote(key, session...)
			if err != nil {
				fmt.Println("Error deleting remote file:", err)
			}

		case "help":
			if len(args) != 2 {
				fmt.Println("Usage: help <command>")
				continue
			}
			fmt.Println(helpText[args[1]])
			continue

		case "quit":
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Unknown command. Supported: store, get, delete, deletelocal, quit")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

package main

import (
	"bufio"
	"strings"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"encoding/csv"
)

const CONFIG_DIR = "/home/hillenr/box/7750/"
var errNotFound = errors.New("not found")

type host struct {
	name     string
	ip       string
	hType    string
	user     string
	password string
	script   string
}

func scanCsvFile(filePath string, hostName string) (host, error) {
	var h = host{}
	f, err := os.Open(CONFIG_DIR + filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
			return h, fmt.Errorf("%q: %w", hostName, errNotFound)
		}
		if err != nil {
			return h, err
		}
		if record[0] == hostName {
			h.name = record[0]
			h.ip = record[1]
			h.hType = record[2]
			h.user = record[3]
			h.password = record[4]
			h.script = record[5]
			return h, nil
		}
	}
}

func scanConfig(str string) string {
	fmt.Print(str)
	config, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	config = strings.Trim(config, "\n")
	return config
}

func main() {
	port := flag.String("port", "22", "SSH port number")
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("Host name argument required")
	}
    hostn := flag.Args()[0]
	h, err := scanCsvFile("hosts.txt", hostn)
	if err != nil {
		usr_host := strings.Split(hostn, "@")
		if len(usr_host) == 2 {
			h.user = usr_host[0]
			h.ip = usr_host[1]
		} else {
			h.user = scanConfig("user: ")
			h.ip = hostn
		}
        h.script = ""
		h.password = scanConfig("password: ")
	}
	client, err := DialWithPasswd(h.ip+":"+*port, h.user, h.password)
	if err != nil {
		log.Fatal("Dial error: ", err)
	}
	defer client.Close()

    var mr io.Reader
    if h.script != "" {
        file, err := os.Open(CONFIG_DIR + h.script)
        if err != nil {
            log.Fatal("File error: ", err)
            os.Exit(1)
        }
        defer file.Close()
        mr = io.MultiReader(file, os.Stdin)
    }
    fmt.Printf("\033]0;%s\007", hostn)
	err = client.Terminal(nil).SetStdio(mr, nil, nil).Start()
	//err = client.Shell().SetStdio(script, &stdout, &stderr).Start()
	if err != nil {
		log.Fatal("Start shell error: ", err)
		os.Exit(1)
	}
    hostn, err = os.Hostname()
    fmt.Printf("\033]0;%s\007", hostn)

	/*
		fmt.Println(stdout.String())
		fmt.Println(stderr.String())
		err = client.Terminal(nil).Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Start terminal error: %v\n", err)
			os.Exit(1)
		}
	*/
}

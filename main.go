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
//	"regexp"
	"time"
    "strconv"

	"encoding/csv"
    "text/tabwriter"
)

const CONFIG_DIR = "/home/hillenr/box/7750/"
const LOG_DIR = "/home/hillenr/docs/nssh_logs/"
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

func printCsvFile(filePath string) (error) {
    w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	f, err := os.Open(CONFIG_DIR + filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
            w.Flush()
            fmt.Printf("\nHost file is at: %s\n", CONFIG_DIR + filePath)
			return nil
		}
		if err != nil {
			return err
		}
        fmt.Fprintf(w, "%s\t\n", strings.Join(record, "\t"))
	}
}

func scanConfig(str string) string {
	fmt.Print(str)
	config, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	config = strings.Trim(config, "\n")
	return config
}

func updateStr(str []byte, pos int, c byte) ([]byte, int) {
    if pos < 0 {
        pos = 0
    }
	for {
		if len(str) > pos {
			str[pos] = c
			pos += 1
			return str, pos
		} else {
            str = append(str, ' ')
        }
	}
}

func scanAnsi(line string, idx int, cursor int) (int, int){
    if line[idx+1] == '[' {  // CSI (control sequence introduced)
        //ansi_str = ansi_str + "["
        digits := ""
        offs := 0
        for i := 2; i < len(line); i++ {
            ac := line[idx + i]
            if ac >= '0' && ac <= '9' {
                digits = digits + string(ac)
            }
            //ansi_str = ansi_str + string(ac)
            if ac >= 0x40 && ac <= 0x7e {
                idx = idx + i
                offs, _ = strconv.Atoi(digits)
                if ac == 'D' { //cursor left
                    cursor -= offs
                }
                if ac == 'C' { //cursor right
                    cursor += offs
                }
                if ac != 'm' { // SGR Set Graphics Rendition
                    //fmt.Printf("%s", ansi_str + ">")
                }
                break
            }
        }
    } else {
        for i := 1; i < len(line); i++ {
            ac := line[idx + i]
            //ansi_str = ansi_str + string(ac)
            if ac >= 0x30 && ac <= 0x7e {
                idx = idx + i
                //fmt.Printf("%s", ansi_str + ">")
                break
            }
        }
    }
    return idx, cursor
}

func fmtLine(line string) []byte {
    output := []byte{}
    cursor := 0
    for i := 0; i < len(line); i++ {
        c := line[i]
        if c == 7 { //bell
            continue
        }
        if c == 13 { //CR
            cursor = 0
            //output = nil
            continue
        }
		if c == 8 && cursor > 0 { //BS
			output, cursor = updateStr(output, cursor-1, ' ')
			cursor -= 1
			//fmt.Print("<BS>")
			continue
		}
        if c == 0x1b { // ESC
            //ansi_str := "<ESC"
            i, cursor = scanAnsi(line, i, cursor)
        } else {
            output, cursor = updateStr(output, cursor, c)
            //fmt.Printf("%c", c)
        }
    }
    //fmt.Println()
    return output
}

func main() {
	port := flag.String("port", "22", "SSH port number")
	printHosts := flag.Bool("h", false, "Print host file")
	flag.Parse()
    if *printHosts {
        printCsvFile("hosts.txt")
		os.Exit(0)
    }
	if len(flag.Args()) == 0 {
        fmt.Println("Host name (or user@IP) required as argument:\n")
        printCsvFile("hosts.txt")
		os.Exit(0)
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
    log_f_n := LOG_DIR + hostn + "_" + time.Now().Format("2006-01-02_150405") + ".log"
    log_f, err := os.OpenFile(log_f_n, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Fatal("File error: ", err)
        os.Exit(1)
    }
    defer log_f.Close()

    r, w := io.Pipe()
	go func(r io.Reader  , w io.Writer, logf *os.File) {
		buf := make([]byte, 1)
        line := []byte{}
        hist := []byte{}
		for {
			_, err := r.Read(buf)
			if err == io.EOF {
				break
			}
            char := buf[0]
            hlen := len(hist)
            if hlen >= 2 && hist[hlen-1] != 13 && char == 10 {
                _, err = w.Write([]byte{13})
                hist = nil
            }
            if char == 10 {
                hist = nil
            }
            hist = append(hist, char)
            _, err = w.Write(buf)
			if err == io.EOF {
				break
			}
            if char < 32 || char > 127 {
                //fmt.Printf("<%X>", char)
            }
            if char == 0 {
                continue
            }
            if char == 10 {
                llen := len(line)
                if llen >= 2 && line[llen-1] == 13 {
                    line = line[:llen-1]
                }
                line = append(line, char)
                //ansi_escape, _ := regexp.Compile(`\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
                //result := ansi_escape.ReplaceAll(line, []byte(""))
                //_, err = logf.WriteString(string(result))
                //_, err = logf.Write(line)
                _, err = logf.Write(fmtLine(string(line)))
                if err == io.EOF {
                    break
                }
                line = nil
                continue
            }
            line = append(line, char)
		}
    }(r, os.Stdout, log_f)
    fmt.Printf("\033]0;%s\007", hostn)
	err = client.Terminal(nil).SetStdio(mr, w, nil).Start()
	if err != nil {
		log.Fatal("Start shell error: ", err)
		os.Exit(1)
	}
    hostn, err = os.Hostname()
    fmt.Printf("\033]0;%s\007", hostn)

}

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
    "strconv"
//	"regexp"

)

const CONFIG_DIR = "/home/hillenr/box/7750/"
const LOG_DIR = "/home/hillenr/docs/nssh_logs/"
var errNotFound = errors.New("not found")

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
            cursor -= 1
            output[cursor] = ' '
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
	flag.Parse()
	if len(flag.Args()) == 0 {
        fmt.Println("Input file required")
		os.Exit(0)
	}
    file, err := os.Open(flag.Args()[0])
    if err != nil {
        log.Fatal("File error: ", err)
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        fmt.Println(string(fmtLine(line)))
        //fmt.Println(line)
    }
}

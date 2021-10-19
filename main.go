package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

var noTrim bool
var isJson bool
var fieldSep string

func main() {
	var parse string
	flag.StringVar(&parse, "p", "all", "The parse expression")

	var output string
	flag.StringVar(&output, "o", "{{ .all }}", "The output expression. It can either be 'json' to output the results as an array of JSON objects, or a Go template which will be used to format the outout.")

	flag.BoolVar(&noTrim, "t", false, "Do not trim whitespace from start and end of parsed values")

	flag.StringVar(&fieldSep, "d", " *", "The default field delimiter")

	var noNewline bool
	flag.BoolVar(&noNewline, "n", false, "Do not emit newline at the end of each output line")

	flag.Parse()
	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var templ *template.Template
	var err error

	if output == "json" {
		isJson = true
	} else {
		templ, err = template.New("main").Parse(output)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	defer func() {
		err := recover()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	// NB: the following line might seem redundant, but the Unquote function also
	// unescapes any escape char provided by the user in the parse format string.
	parse, _ = strconv.Unquote(`"` + parse + `"`)

	choppers := Parse(parse)
	var values map[string]string = make(map[string]string)

	var file *os.File
	if flag.NArg() > 0 {
		file, err = os.Open(flag.Arg(0))
		defer file.Close()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		file = os.Stdin
	}

	in := bufio.NewScanner(file)
	in.Split(bufio.ScanLines)

	if isJson {
		fmt.Println("[")
	}
	var line int = 0
	for in.Scan() {
		text := []byte(in.Text())
		next := 0

		for k := range values {
			delete(values, k)
		}

		for _, chopper := range choppers {
			next = chopper.Chop(text, next, values)
		}

		if isJson {
			if line > 0 {
				fmt.Println(",")
			}
			line++
			if data, err := json.Marshal(values); err == nil {
				fmt.Printf("  %s", string(data))
			}

		} else {
			templ.Execute(os.Stdout, values)
			if !noNewline {
				fmt.Println("")
			}
		}
	}
	if isJson {
		fmt.Printf("\n]\n")
	}

}

func Text(text []byte) string {
	if !noTrim {
		return strings.Trim(string(text), " ")
	} else {
		return string(text)
	}
}

// A Chopper is a thing that extracts a chunk of test from the input, starting at 'start',
// up to either an absolute position in the input, a relative position from start, or the next instance of a text delimiter,
// and saves it under a name in the 'values' map (unless the name is '.', in which case it is dropped).
type Chopper interface {
	Chop(input []byte, start int, values map[string]string) int
}

// Extracts up to an absolute position
type AbsChunk struct {
	name  string
	until int
}

func (ac *AbsChunk) Chop(input []byte, start int, values map[string]string) int {
	if ac.name != "." {
		values[ac.name] = Text(input[start:ac.until])
	}
	return ac.until
}

// Extracts up to a relative position
type RelChunk struct {
	name  string
	until int
}

func (rc *RelChunk) Chop(input []byte, start int, values map[string]string) int {
	if rc.name != "." {
		values[rc.name] = Text(input[start : start+rc.until])
	}
	return start + rc.until
}

// Extracts up to the next occurrence of the delimiter
// If delimiter text ends in '*', also skips over any subsequent instances
// of delimiter text
type DelimChunk struct {
	name    string
	until   string
	skipAll bool
}

func (dc *DelimChunk) Chop(input []byte, start int, values map[string]string) int {
	var text []byte
	var end int

	if dc.until == "" {
		text = input[start:]
		end = start + len(text)

	} else {

		endText := bytes.Index(input[start:], []byte(dc.until))

		if endText == -1 {
			text = input[start:]
			end = start + len(text)

		} else {
			text = input[start : start+endText]
			var delimLen int = len(dc.until)
			end = start + len(text) + delimLen

			if dc.skipAll {
				for end < len(input) && string(input[end:end+delimLen]) == dc.until {
					end += delimLen
				}
			}
		}
	}

	if dc.name != "." {
		values[dc.name] = Text(text)
	}

	return end
}

// A version of the Go stdlib ScanWords impl, that also returns blocks of text within single & double quotes
func ScanText(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	start := 0
	var r rune
	for width := 0; start < len(data); start += width {
		r, width = utf8.DecodeRune(data[start:])
		if !unicode.IsSpace(r) {
			break
		}
	}

	// Save whether text start with a quote
	var quoteChar rune
	var quoteStart int
	if r == '\'' || r == '"' {
		quoteChar = r
		quoteStart = start
		start++
	}

	// Scan until space or next quote, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		r, width = utf8.DecodeRune(data[i:])
		if quoteChar > 0 && r == quoteChar {
			return i + width + 1, data[quoteStart : i+1], nil

		} else if quoteChar == 0 && unicode.IsSpace(r) {
			return i + width, data[start:i], nil
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}

func CreateDelimChunk(name, until string) *DelimChunk {
	var skipAll bool = false
	if strings.HasSuffix(until, "*") && !strings.HasSuffix(until, `\*`) {
		until = until[:len(until)-1]
		skipAll = true
	}
	return &DelimChunk{
		name:    name,
		until:   until,
		skipAll: skipAll,
	}

}

func Parse(parse string) []Chopper {
	var choppers []Chopper

	scanner := bufio.NewScanner(strings.NewReader(parse))
	scanner.Split(ScanText)

	var name string = ""

	for true {
		// Read item if we don't have it already
		if name == "" {
			if scanner.Scan() {
				name = scanner.Text()
			} else {
				break // EOL
			}
		}

		var next string
		if scanner.Scan() {
			next = scanner.Text()
		} else {
			// EOL so add chopper to save the remainder of input under the current name
			choppers = append(choppers, CreateDelimChunk(name, ""))
			break
		}

		// Non-numeric tokens first
		if numVal, err := strconv.Atoi(next); err != nil {

			// If quoted text, add chopper to delimit input at next instance of the text
			if (next[0] == '"' && next[len(next)-1] == '"') ||
				(next[0] == '\'' && next[len(next)-1] == '\'') {
				choppers = append(choppers, CreateDelimChunk(name, strings.Trim(next, `'"`)))
				name = ""

			} else {
				// Otherwise, it is another name. Add chopper to delimit at the next default field separator
				// and save the text just read as the current name for the next time round
				choppers = append(choppers, CreateDelimChunk(name, fieldSep))
				name = next

			}

			// Numeric delimiters
		} else {

			// If signed, add relative chopper
			if next[0] == '+' || next[0] == '-' {

				choppers = append(choppers, &RelChunk{
					name:  name,
					until: numVal,
				})
				name = ""

				// Otherwise absolute chopper
			} else {

				choppers = append(choppers, &AbsChunk{
					name:  name,
					until: numVal,
				})
				name = ""

			}
		}
	}

	return choppers
}

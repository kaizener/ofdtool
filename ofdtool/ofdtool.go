package main

import (
	"bufio"
	"fmt"
	"github.com/axgle/mahonia"
	"github.com/kaizener/ofdfielddefine"
	"gopkg.in/alecthomas/kingpin.v2"
	. "github.com/logrusorgru/aurora"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type fileInfo struct {
	sender       string
	receiver     string
	transferDate string
	fieldCnt     int
	fieldNames   []string
	rowCnt       int
	rowInf       []fieldInf
}

type fieldInf struct {
	bi      int
	ei      int
	name    string
	comment string
	typ     string
	format  string
	size    int
	scale   int
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var (
	app = kingpin.New("ofdtool", "A OFD file(开放式基金数据文件) tool.")

	parse = app.Command("parse", "parse a file.")
	f     = parse.Arg("file", "OFD file(eg. OFD_27_378_20170417_07.TXT)").Required().String()
	q     = parse.Flag("q", "filter value(eg. --q=270004)").PlaceHolder("FILTER-VALUE").String()
	t     = parse.Flag("t", "show type info").Default("false").Bool()
	dc    = parse.Flag("dc", "disable colorful").Default("false").Bool()

	modify = app.Command("modify", "modify a file.")
	of     = modify.Arg("original file", "OFD file(eg. OFD_27_378_20170417_07.TXT)").Required().String()

	diff = app.Command("diff", "diff two files.")
	lf   = diff.Arg("lside file", "OFD file(eg. OFD_27_378_20170417_07.TXT)").Required().String()
	rf   = diff.Arg("rside file", "OFD file(eg. OFD_27_378_20170417_07.TXT)").Required().String()

	decoder = mahonia.NewDecoder("GBK")
)

func main() {
	kingpin.Version("0.6.0")
	kingpin.CommandLine.HelpFlag.Short('h')

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case parse.FullCommand():
		parseCmd()
	case modify.FullCommand():
		modifyCmd()
	case diff.FullCommand():
		diffCmd()
	}
}

func modifyCmd() {
	fmt.Println("TBD")
}

func diffCmd() {
	fmt.Println("TBD")
}

func parseCmd() {
	start := time.Now()
	if *f == "" {
		log.Fatal("must specify OFD file, -f <filename>")
	}
	var filter bool
	if *q != "" {
		fmt.Println(fmt.Sprintf("use [%s] filter...", *q))
		filter = true
	}
	var matched int

	df, err := os.Open(*f)
	checkError(err)
	defer df.Close()

	scanner := bufio.NewScanner(df)
	checkError(err)

	lineSep := fmt.Sprintf("  %s", strings.Repeat("#", 78))

	var fi fileInfo
	var bi int
	var v string

	ln := 1
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case ln == 1:
			fmt.Println("Header:")
			fmt.Println(fmt.Sprintf("  [%s]", trimRSpace(line)))
			fmt.Println("Meta:")
		case ln == 2:
		case ln == 3:
			fi.sender = trimRSpace(line)
			fmt.Println(fmt.Sprintf("  sender:[%s]", fi.sender))
		case ln == 4:
			fi.receiver = trimRSpace(line)
			fmt.Println(fmt.Sprintf("  receiver:[%s]", fi.receiver))
		case ln == 5:
			fi.transferDate = trimRSpace(line)
			fmt.Println(fmt.Sprintf("  transferDate:[%s]", fi.transferDate))
		case ln < 10:
		case ln == 10:
			fi.fieldCnt, err = strconv.Atoi(strings.TrimPrefix(trimRSpace(line), "0"))
			checkError(err)
			fmt.Println(fmt.Sprintf("  fieldCnt:[%d]", fi.fieldCnt))
		case ln <= 10+fi.fieldCnt:
			fi.fieldNames = append(fi.fieldNames, strings.ToLower(trimRSpace(line)))
		case ln == 10+fi.fieldCnt+1:
			fi.rowCnt, err = strconv.Atoi(strings.TrimPrefix(trimRSpace(line), "0"))
			checkError(err)
			if fi.rowCnt > 0 {
				for _, v := range fi.fieldNames {
					o := ofdfielddefine.Define[v]
					checkError(err)
					switch m := o.(type) {
					case map[string]interface{}:
						typ := m["type"].(string)
						size := m["size"].(int)
						var scale int
						if typ == "N" {
							scale = m["scale"].(int)
						}
						fi.rowInf = append(fi.rowInf, fieldInf{bi: int(bi), ei: int(bi + size), name: v, typ: typ, comment: m["comment"].(string), size: size, scale: scale, format: fmt.Sprintf("%%.%df", int(scale))})
						bi += size
					default:
					}
				}
				fmt.Println("Rows:")
			}
		case ln <= 10+fi.fieldCnt+1+fi.rowCnt:
			rm := false
			pl := ""
			gbkBytes := []byte(line)
			for _, ri := range fi.rowInf {
				subBytes := gbkBytes[ri.bi:ri.ei]
				switch {
				case ri.typ == "A":
					v = trimRSpace(string(subBytes))
				case ri.typ == "C":
					v = trimRSpace(decoder.ConvertString(string(subBytes)))
				case ri.typ == "N":
					dividend, _ := strconv.ParseFloat(string(subBytes), 64)
					divisor := math.Pow10(int(ri.scale))
					v = fmt.Sprintf(ri.format, dividend/divisor/1.0)
				}

				if filter && v == *q {
					rm = true
					pl += colorful(true, ri.typ, ri.size, ri.scale, ri.name, v, ri.comment)
				} else {
					pl += colorful(false, ri.typ, ri.size, ri.scale, ri.name, v, ri.comment)
				}
			}
			if !filter {
				fmt.Println(lineSep)
				fmt.Print(pl)
			} else if filter && rm {
				matched++
				fmt.Println(lineSep)
				fmt.Println(pl)
			}

		case ln == 10+fi.fieldCnt+1+fi.rowCnt+1:
			fmt.Println("Footer:")
			fmt.Println(fmt.Sprintf("  [%s]", trimRSpace(line)))

		}
		ln++
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if filter {
		log.Printf("total:[%d],matched:[%d] elapsed %s", fi.rowCnt, matched, elapsed)
	} else {
		log.Printf("total:[%d],elapsed %s", fi.rowCnt, elapsed)
	}
}

func colorful(match bool, typ string, size, scale int, name, value, comment string) string {
	var cn, cc, cv, cti interface{} = name, comment, value, ""
	if *t {
		if typ == "A" || typ == "C" {
			cti = fmt.Sprintf("%s(%d)", typ, size)
		} else if scale != 0 {
			cti = fmt.Sprintf("%s(%d,%d)", typ, size, scale)
		} else {
			cti = fmt.Sprintf("%s(%d)", typ, size)
		}
	}
	switch *dc {
	case false:
		cn, cc, cti = Cyan(name), Gray(cc), Gray(cti)
		if match {
			cv = Red(value)
		} else if typ == "A" || typ == "C" {
			cv = Green(value)
		} else {
			cv = Magenta(value)
		}
	default:
	}
	if *t {
		return fmt.Sprintf("  %s:[%s]#%s:%s\n", cn, cv, cc, cti)
	} else {
		return fmt.Sprintf("  %s:[%s]#%s\n", cn, cv, cc)
	}
}

func trimRSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

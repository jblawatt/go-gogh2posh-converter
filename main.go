package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/go-ini/ini"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// https://technet.microsoft.com/en-us/library/cc957408.aspx
// https://github.com/Mayccoll/Gogh
// https://github.com/mbadolato/iTerm2-Color-Schemes

// Color
// 0 - Black - x
// 1 - Blue
// 2 - Green - x (Git Diff)
// 3 - Aqua - x (Literal) (ProgressBackgroundColor)
// 4 - Red - x
// 5 - Purple /// - x (Text Color)
// 6 - Yellow - x
// 7 - White
// 8 - Gray - x (Parameter) (Sign)
// 9 - Light Blue
// A - Light Green - x (Reserved Word)
// B - Light Aqua
// C - Light Red - x (Error (Red))
// D - Light Purple
// E - Light Yellow - x (Command / Warning)
// F - Bright White

// $ (Get-Host).PrivateData
// ErrorForegroundColor    : Red
// ErrorBackgroundColor    : Black
// WarningForegroundColor  : Yellow
// WarningBackgroundColor  : Black
// DebugForegroundColor    : Yellow
// DebugBackgroundColor    : Black
// VerboseForegroundColor  : Yellow
// VerboseBackgroundColor  : Black
// ProgressForegroundColor : Yellow
// ProgressBackgroundColor : DarkCyan

// PSColors is the type where the options are parsed in.
type PSColors struct {
	ColorTable00 string
	ColorTable01 string
	ColorTable02 string
	ColorTable03 string
	ColorTable04 string
	ColorTable05 string
	ColorTable06 string
	ColorTable07 string
	ColorTable08 string

	ColorTable09 string
	ColorTable10 string
	ColorTable11 string
	ColorTable12 string
	ColorTable13 string
	ColorTable14 string
	ColorTable15 string

	ScreenColors string
	PopupColors  string
}

// SetValue ...
func (p *PSColors) SetValue(attribute string, value string) {
	reflect.ValueOf(p).Elem().FieldByName(attribute).SetString(value)
}

// GetValue ...
func (p *PSColors) GetValue(field string) string {
	return reflect.ValueOf(p).Elem().FieldByName(field).String()
}

func createRegFileContent(colors PSColors) (string, error) {
	var buffer bytes.Buffer
	temp := template.New("template")
	temp.Parse(`Windows Registry Editor Version 5.00
; generated file
[HKEY_CURRENT_USER\Console]
"ColorTable00"=dword:{{.ColorTable00}}
"ColorTable01"=dword:{{.ColorTable01}}
"ColorTable02"=dword:{{.ColorTable02}}
"ColorTable03"=dword:{{.ColorTable03}}
"ColorTable04"=dword:{{.ColorTable04}}
"ColorTable05"=dword:{{.ColorTable05}}
"ColorTable06"=dword:{{.ColorTable06}}
"ColorTable07"=dword:{{.ColorTable07}}

"ColorTable08"=dword:{{.ColorTable08}}
"ColorTable09"=dword:{{.ColorTable09}}
"ColorTable10"=dword:{{.ColorTable10}}
"ColorTable11"=dword:{{.ColorTable11}}
"ColorTable12"=dword:{{.ColorTable12}}
"ColorTable13"=dword:{{.ColorTable13}}
"ColorTable14"=dword:{{.ColorTable14}}
"ColorTable15"=dword:{{.ColorTable15}}

"ScreenColors"=dword:{{.ScreenColors}}
"PopupColors"=dword:{{.PopupColors}}`)

	temp.Execute(&buffer, colors)
	return buffer.String(), nil
}

func dwordFromHex(hex string) string {
	v := strings.Join([]string{
		"00",
		hex[4:6],
		hex[2:4],
		hex[0:2],
	}, "")
	return strings.ToUpper(v)
}

type GoghExtractor struct{}

func (e *GoghExtractor) Extract(in io.Reader, fgColorIndex int, bgColorIndex int) PSColors {

	scanner := bufio.NewScanner(in)

	colorRegex, _ := regexp.Compile(`COLOR_(\d{2})="#(\w{6})`)
	foregroundRegex, _ := regexp.Compile(`FOREGROUND_COLOR="#(\w{6})"`)
	backgroundRegex, _ := regexp.Compile(`BACKGROUND_COLOR="#(\w{6})"`)

	colors := PSColors{}
	var fgValue string
	var bgValue string

	for scanner.Scan() {
		text := scanner.Text()
		if colorRegex.MatchString(text) {
			matchesArr := colorRegex.FindStringSubmatch(text)
			value := dwordFromHex(matchesArr[2])
			valueInt, _ := strconv.Atoi(matchesArr[1])
			valueInt--
			key := padLeft(strconv.Itoa(valueInt), "0", 2)
			colors.SetValue("ColorTable"+key, strings.ToUpper(value))
		}
		if foregroundRegex.MatchString(text) {
			matchesArr := foregroundRegex.FindStringSubmatch(text)
			fgValue = matchesArr[1]
		}
		if backgroundRegex.MatchString(text) {
			matchesArr := backgroundRegex.FindStringSubmatch(text)
			bgValue = matchesArr[1]
		}
	}

	fgIndex := ""
	for i := 0; i < 16; i++ {
		key := "ColorTable" + padLeft(strconv.Itoa(i), "0", 2)
		if colors.GetValue(key) == fgValue {
			fgIndex = strconv.FormatInt(int64(i), 16)
			break
		}
	}

	if fgIndex == "" {
		colors.SetValue("ColorTable"+padLeft(strconv.FormatInt(int64(fgColorIndex), 10), "0", 2), dwordFromHex(fgValue))
		fgIndex = strconv.FormatInt(int64(fgColorIndex), 16)
	}

	bgIndex := ""
	for i := 0; i < 16; i++ {
		key := "ColorsTable" + padLeft(strconv.Itoa(i), "0", 2)
		if colors.GetValue(key) == bgValue {
			bgIndex = padLeft(strconv.FormatInt(int64(i), 16), "0", 4)
			break
		}
	}

	if bgIndex == "" {
		colors.SetValue("ColorTable"+padLeft(strconv.FormatInt(int64(bgColorIndex), 10), "0", 2), dwordFromHex(bgValue))
		bgIndex = strconv.FormatInt(int64(bgColorIndex), 16)
	}

	colors.SetValue("ScreenColors", padLeft(bgIndex+fgIndex, "0", 8))
	colors.SetValue("PopupColors", padLeft(fgIndex+bgIndex, "0", 8))

	return colors

}

func padLeft(str, pad string, lenght int) string {
	for {
		if len(str) == lenght {
			return str[0:lenght]
		}
		str = pad + str
	}
}

type Extractor interface {
	Extract(in io.Reader, fgColorIndex int, bgColorIndex int) PSColors
}

type KonsoleExtractor struct{}

func (e *KonsoleExtractor) Extract(in io.Reader, fgColorIndex int, bgColorIndex int) PSColors {
	colors := PSColors{}
	iniSource, _ := ini.Load(in)
	for i := 0; i < 8; i++ {
		for _, suff := range []string{"", "Intense"} {
			sectionKey := "Color" + strconv.FormatInt(int64(i), 10) + suff
			sec, _ := iniSource.GetSection(sectionKey)
			key, _ := sec.GetKey("Color")
			val := key.String()
			parts := strings.Split(val, ",")
			a, _ := strconv.Atoi(parts[0])
			b, _ := strconv.Atoi(parts[1])
			c, _ := strconv.Atoi(parts[2])

			hexVal := hex.EncodeToString([]byte{
				byte(a),
				byte(b),
				byte(c),
			})
			var v int
			if suff == "Intense" {
				v = i + 8
			} else {
				v = i
			}

			colors.SetValue(
				"ColorTable"+padLeft(strconv.FormatInt(int64(v), 10), "0", 2),
				dwordFromHex(hexVal))
		}
	}

	var sec *ini.Section
	var key *ini.Key
	var parts []string
	sec, _ = iniSource.GetSection("Foreground")
	key, _ = sec.GetKey("Color")
	parts = strings.Split(key.String(), ",")

	var a, _ = strconv.Atoi(parts[0])
	var b, _ = strconv.Atoi(parts[1])
	var c, _ = strconv.Atoi(parts[2])

	hexVal := hex.EncodeToString([]byte{
		byte(a),
		byte(b),
		byte(c),
	})

	colors.SetValue("ColorTable"+padLeft(strconv.FormatInt(int64(fgColorIndex), 10), "0", 2), dwordFromHex(hexVal))

	sec, _ = iniSource.GetSection("Background")
	key, _ = sec.GetKey("Color")
	parts = strings.Split(key.String(), ",")
	a, _ = strconv.Atoi(parts[0])
	b, _ = strconv.Atoi(parts[1])
	c, _ = strconv.Atoi(parts[2])

	hexVal = hex.EncodeToString([]byte{
		byte(a),
		byte(b),
		byte(c),
	})

	colors.SetValue("ColorTable"+padLeft(strconv.FormatInt(int64(bgColorIndex), 10), "0", 2), dwordFromHex(hexVal))

	fgIndex := strconv.FormatInt(int64(fgColorIndex), 16)
	bgIndex := strconv.FormatInt(int64(bgColorIndex), 16)

	colors.SetValue("ScreenColors", padLeft(bgIndex+fgIndex, "0", 8))
	colors.SetValue("PopupColors", padLeft(fgIndex+bgIndex, "0", 8))

	return colors

}

func main() {
	var inFile string
	var outFile string
	var inURL string
	var logFile string
	var goghTheme string

	var fgColorTableIndex int
	var bgColorTableIndex int
	var extractorKey string

	flag.StringVar(&inFile, "inFile", "", "die datei die geparsed werden soll.")
	flag.StringVar(&outFile, "out", "", "Ausgabedatei. Default os.Stdout")
	flag.StringVar(&inURL, "inURL", "", "Load From URL https://mayccoll.github.io/Gogh/")
	flag.StringVar(&logFile, "logFile", "", "Log File")
	flag.StringVar(&goghTheme, "goghTheme", "", "Gogh Theme. Will be loaded from the internet.")
	flag.IntVar(&fgColorTableIndex, "fgColorIndex", 1, "Foreground color table index.")
	flag.IntVar(&bgColorTableIndex, "bgColorIndex", 4, "Foreground color table index.")
	flag.StringVar(&extractorKey, "extractor", "gogh", "...")

	flag.Parse()

	extractors := map[string]Extractor{
		"gogh":    &GoghExtractor{},
		"konsole": &KonsoleExtractor{},
	}

	if logFile != "" {
		logWriter, _ := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		defer logWriter.Close()
		log.SetOutput(logWriter)
	}

	if goghTheme != "" {
		inURL = strings.Join([]string{
			"https://raw.githubusercontent.com/Mayccoll/Gogh/master/themes/",
			goghTheme,
			".sh",
		}, "")
	}

	if inFile == "" && inURL == "" {
		flag.Usage()
		return
	}

	var outWriter io.Writer

	if outFile == "" {
		outWriter = os.Stdout
	} else {
		outFileHandler, _ := os.OpenFile(outFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		defer outFileHandler.Close()
		outWriter = outFileHandler
	}

	var inReader io.Reader

	if inFile != "" {
		inReader, _ = os.Open(inFile)
	} else {
		httpResp, _ := http.Get(inURL)
		inReader = httpResp.Body
	}

	colors := extractors[extractorKey].Extract(inReader, fgColorTableIndex, bgColorTableIndex)
	regContent, _ := createRegFileContent(colors)

	fmt.Fprint(outWriter, regContent)

}

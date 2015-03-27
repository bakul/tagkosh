package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

var (
	flagv   = flag.Bool("v", false, "print verbose output")
	flagf   = flag.String("f", "", "file containing names of files to process")
	verbose bool

	spaceRE = regexp.MustCompile("[ \t]+")
	lineRE  = regexp.MustCompile("\r?\n")

	cat = make(map[string]string, 10)
)

func init() {
	abbr := []string{
		"અ૦", "અવ્યય",
		"અ૦ક્રિ૦", "અકર્મક ક્રિયાપદ",
		"ઉદા૦", "ઉદાહરણ",
		"અે૦વ૦", "અેકવચન",
		"કૃ૦", "કૃદંત",
		"ક્રિ૦", "ક્રિયાપદ",
		"ન૦", "નપુંસક લિંગ",
		"ન૦બ૦વ૦", "નપુંસક લિંગ, બહુવચન",
		"પું૦", "પુંલિંગ",
		"પું૦બ૦વ૦", "પુંલિંગ, બહુવચન",
		"પ્રા૦વિ૦", "પ્રાણી વિજ્ઞાન",
		"બ૦વ૦", "બહુવચન",
		"ભ૦કા૦", "ભવિષ્યકાળ",
		"ભ૦કૃ૦", "ભવિષ્યકૃદંત",
		"ભૂ૦કા૦", "ભૂતકાળ",
		"ભૂ૦કૃ૦", "ભૂતકૃદંત",
		"રવ૦", "રવાનુકારી",
		"ર૦વિ૦", "રસાયણ વિજ્ઞાન",
		"વ૦કા૦", "વર્તમાનકાળ",
		"વ૦કૃ૦", "વર્તમાનકૃદંત",
		"વ૦વિ૦", "વનસ્પતિ વિજ્ઞાન",
		"વિ૦", "વિશેષણ",
		"વિ૦ન૦", "વિશેષણ, નપુંસક લિંગ",
		"વિ૦પું૦", "વિશેષણ, પુંલિંગ",
		"વિ૦સ્ત્રી૦", "વિશેષણ, સ્ત્રીલિંગ",
		"શ૦પ્ર૦", "શબ્દપ્રયોગ",
		"શ૦વિ૦", "શરીર વિજ્ઞાન",
		"સ૦", "સર્વનામ",
		"સર૦", "સરખાવો",
		"સા૦કૃ૦", "સામાન્ય કૃદંત",
		"સ્ત્રી૦", "સ્ત્રીલિંગ",
		"સ્ત્રી૦બ૦વ૦", "સ્ત્રીલિંગ, બહુવચન",
	}
	for i := 0; i < len(abbr); i += 2 {
		cat[abbr[i]] = abbr[i+1]
	}

}

var state = 0
type Word struct {
     cat  string
}

var words = make(map[string]*Word, 10)

func processfile(file string, dst *bufio.Writer) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %v\n", file, err)
		return
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "processingv: %v\n", file)
	}

	s := string(data)
	if s[len(s)-1] != '\n' {
		s += "\n"
	}
	lines := lineRE.Split(s, -1)
	if verbose {
		fmt.Fprintf(os.Stderr, "%d lines\n", len(lines)-1)
	}
	for i := 0; i < len(lines); i++ {
		l := strings.TrimSpace(lines[i])
		if len(l) == 0 || l[0] == '#' {
			continue
		}
		w := spaceRE.Split(l, -1)
		switch state {
		case 0:
			if _, ok := words[w[0]]; ok {
		     	      fmt.Fprintf(os.Stderr, "Duplicate word: %v\n", w[0])
			}
			words[w[0]] = &Word{cat:w[1]}
		case 1:
		default:
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "%d: %d words\n", i+1, len(w))
		}
	}
}

func process(src []string, dst string) {
	if path.Ext(dst) != ".tag" {
		fmt.Fprintf(os.Stderr, "dst file must have .tag extension\n")
		os.Exit(1)
	}
	d, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %v\n", dst, err)
		return
	}
	defer d.Close()
	w := bufio.NewWriter(d)
	for _, file := range src {
		processfile(file, w)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: tagkosh [-v] input-file [input-file ...] output\n"+
			"       tagkosh [-v] input-dir output\n"+
			"       tagkosh [-v] -f input-list output\n")
	os.Exit(1)
}

func readlines(listfile string) ([]string, error) {
	var lines []string
	f, err := os.Open(listfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		l, err := r.ReadString('\n')
		if err != nil {
			break
		}
		l = strings.TrimSpace(l)
		if len(l) == 0 || l[0] == '#' {
			continue
		}
		lines = append(lines, l)
	}
	return lines, nil
}

func readdir(name string) ([]string, error) {
	var lines []string
	f, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if f.Mode().IsRegular() {
		return []string{name}, nil
	}
	d, err := ioutil.ReadDir(name)
	if err != nil {
		return nil, err
	}
	for _, f := range d {
		if !f.Mode().IsRegular() {
			continue
		}
		lines = append(lines, name+"/"+f.Name())
	}
	return lines, err
}

func main() {
	var err error
	var dst string
	var src []string
	var n int

	flag.Parse()
	verbose = *flagv
	switch n = flag.NArg(); n {
	case 0:
		usage()
	case 1:
		if *flagf == "" {
			usage()
		}
		src, err = readlines(*flagf)
		if err != nil {
			log.Panicf("error: %v\n", err)
		}
	case 2:
		src, err = readdir(flag.Arg(0))
		if err != nil {
			log.Panicf("error: %v\n", err)
		}
	default:
		src = make([]string, n-1)
		for i := 0; i < n-1; i++ {
			src[i] = flag.Arg(i)
		}
	}
	dst = flag.Arg(n - 1)
	process(src, dst)
}

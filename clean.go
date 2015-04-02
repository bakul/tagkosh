package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

const (
	X = byte(iota)
	V
	C
	M
	H
	D
)

var (
	flagd = flag.Bool("d", false, "print debugging output")
	flagf = flag.Int("f", 0, "field to expand")
	verbose bool

	spaceRE = regexp.MustCompile("[ \t]+")
	semiRE = regexp.MustCompile(";")
	lineRE  = regexp.MustCompile("\r?\n")

	cat = make(map[string]string, 10)
	cc =  [] struct { kind byte; low, high rune } {
		{V, 0xa85, 0xa94}, // vowel
		{C, 0xa95, 0xab9}, // consonant
		{M, 0xabe, 0xacc}, // Matra (vowel sign)
		{H, 0xacd, 0xaac}, // halant or virama
		{D, 0xa81, 0xa83}, // diacritic: anusvar,chandrabindu,visarga
	}
	runeType = make(map[rune]byte, 256)
)

func init() {
	for _, c := range cc {
		for k := c.low ; k <= c.high; k++ {
			runeType[k] = c.kind
		}
	}
}

// Askhar rules:
//	X V VD C(HC)*M?D?

// State machine:
//	s0 X s0/
//	s0 V s1 D s0/
//           s1/
//	s0 C s2 H s3 C s2
//	          s3/
//	     s2 M s4 D s0/
//	          s4/
//	     s2 D s0/
//	     s2/
//
// state sN/ indicates anything else terminates the akshar and revert to s0
//
func aksharize(word string) (ix []int) {
	state := 0
        fmt.Printf("\nlen=%d ", len(word))
	for i, r := range word {
		fmt.Printf("i=%d r=%x %d->", i, r, state)
		k := runeType[r]
		switch state {
		case 0:
			switch k {
			case X: state = 0
			case V: state = 1
			case C: state = 2
			default: continue // error
			}
		case 1:
			switch k {
			case D: state = 0; continue
			case X: state = 0
			case V: state = 1
			case C: state = 2
			default:state = 0 // error
			}
		case 2:
			switch k {
			case H: state = 3; continue
			case M: state = 4; continue
			case D: state = 0; continue
			case X: state = 0
			case V: state = 1
			case C: state = 2
			default:state = 0 // error
			}
		case 3:
			switch k {
			case C: state = 2; continue
			case X: state = 0 
			case V: state = 1 // error
			default:state = 0 // error
			}
		case 4:
			switch k {
			case D: state = 0; continue
			case X: state = 0
			case V: state = 1
			case C: state = 2
			default:state = 0 // error
			}
		}
        	fmt.Printf("%d ", state)
		ix = append(ix, i)
	}
        fmt.Printf("ix=%v\n", ix)
	return ix
}

func lastIndex(word string) int {
	ix := aksharize(word)
	if ix == nil {
		return -1
	}
	return ix[len(ix)-1]
}

func dump(word string) {
	fmt.Printf("%d", len(word))
	s:='('
	for _,r:=range word {
		fmt.Printf("%c%x", s, r)
		s=' '
	}
	fmt.Printf(")")
}

/*
words = mword {"," mword}
mword = word ["(" mods ")"]
mods = mod {"," mod}
mod = ("+" | "-") word
*/
func expand(field string)(words []string) {
	word := ""
	word1 := ""
	op := ' '
	partial := false
	off := 0
	state := 0
	foo:
	for i, r := range field+"|" {
		fmt.Printf("%d ", i)
		switch r {
		case '(':
			if partial {
				word = field[off:i]
				word1 = word[0:lastIndex(word)]
				partial = false
				fmt.Printf("word:")
				dump(word)
				words = append(words, word)
			}
			state = 1
		case ')', ',', ' ', '|':
			if partial {
				last := field[off:i]
				if state == 0 {
					word = last
				} else {
					fmt.Printf("last:%c",op)
					dump(last)
					switch op {
					case '+': last = word + last 
					case '-': last = word1 + last
					}
				}
			        fmt.Printf("->")
				dump(last)
				words = append(words, last)
				partial = false
			}
			if r == '|' {
				break foo
			}
			if r == ')' {
				state = 0
			}
		case '+','-':
			op = r
		default:
			if !partial {
				partial = true
				off = i
			}
			continue
		}
		off = i + 1
	}
	fmt.Printf("->%d words\n", len(words))
	return words
}

func processfile(file string, dst *bufio.Writer) {
	field := *flagf
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
		w := semiRE.Split(l, -1)
		// Format:
		// 1;;words,....
		for i, word := range expand(w[field]) {
			fmt.Printf("%d: ", i);dump(word)
			fmt.Printf("\n")
			fmt.Fprintf(dst, "%s;%s\n", word, l)
		}
	}
}

func process(src []string, dst string) {
	d, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %v\n", dst, err)
		return
	}
	defer d.Close()
	w := bufio.NewWriter(d)
	defer w.Flush()
	for _, file := range src {
		processfile(file, w)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: clean [-v -f field] input-file output\n")
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
	var dst string
	var src []string
	var n int

	flag.Parse()
	verbose = *flagd
	switch n = flag.NArg(); n {
	case 0, 1:
		usage()
	default:
		src = make([]string, n-1)
		for i := 0; i < n-1; i++ {
			src[i] = flag.Arg(i)
		}
	}
	dst = flag.Arg(n - 1)
	process(src, dst)
}

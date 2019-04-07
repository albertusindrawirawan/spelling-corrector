package spelling_corrector

import (
	"io/ioutil"
	"math"
	"regexp"
	"strings"
)

type Corrector struct {
	model   map[string]int
	maxChar int
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"

type Split struct {
	l string
	r string
}

func New(model string) *Corrector {
	var mc float64

	words := make(map[string]int)
	pattern := regexp.MustCompile("[a-z]+")

	if content, err := ioutil.ReadFile(model); err == nil {
		for _, w := range pattern.FindAllString(strings.ToLower(string(content)), -1) {
			words[w]++
			mc = math.Max(mc, float64(len(w)))
		}
	} else {
		panic("Failed to load training data")
	}
	return &Corrector{model: words, maxChar: int(mc)}
}

func (c *Corrector) edits1(word string, ch chan string) {
	var splits []Split
	for i := 0; i < len(word)+1; i++ {
		splits = append(splits, Split{word[:i], word[i:]})
	}

	for _, s := range splits {
		if len(s.r) > 0 {
			ch <- s.l + s.r[1:]
		}
		if len(s.r) > 1 {
			ch <- s.l + string(s.r[1]) + string(s.r[0]) + s.r[2:]
		}
		for _, c := range alphabet {
			if len(s.r) > 0 {
				ch <- s.l + string(c) + s.r[1:]
			}
		}
		for _, c := range alphabet {
			ch <- s.l + string(c) + s.r
		}
	}
}

func (c *Corrector) edits2(word string, ch chan string) {
	ch1 := make(chan string, 1024*1024)
	go func() { c.edits1(word, ch1); ch1 <- "" }()
	for e1 := range ch1 {
		if e1 == "" {
			break
		}
		c.edits1(e1, ch)
	}
}

func (c *Corrector) best(word string, edits func(string, chan string), model map[string]int) (int, string) {
	ch := make(chan string, 1024*1024)
	go func() { edits(word, ch); ch <- "" }()
	maxFreq := 0
	correction := ""
	for word := range ch {
		if word == "" {
			break
		}
		if freq, present := model[word]; present && freq > maxFreq {
			maxFreq, correction = freq, word
		}
	}
	return maxFreq, correction
}

func (c *Corrector) Correct(word string) string {
	if len(word) > c.maxChar {
		return  word
	}
	if _, present := c.model[word]; present {
		return word
	}

	max1, correction1 := c.best(word, c.edits1, c.model)
	max2, correction2 := c.best(word, c.edits2, c.model)

	if max1 == 0 && max2 == 0 {
		return word
	}

	if max2 > max1 {
		return correction2
	}

	return correction1
}

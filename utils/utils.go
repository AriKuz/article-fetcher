package utils

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

// WriteStringToFile writes a string into a file with a chosen name
func WriteStringToFile(name, content string) error {
	if err := os.WriteFile(name, []byte(content), 0644); err != nil {
		log.Error().Msgf("writeStringToFile failed: %v", err)
		return err
	}
	return nil
}

// ReadLinesFromFile reads a file line by line and appends to a []string
func ReadLinesFromFile(fileName string, isBank bool, bankOfWords map[string]struct{}, alphaRegexp *regexp.Regexp) ([]string, error) {
	var lines []string
	// open the file for reading
	file, err := os.Open(fileName)
	if err != nil {
		log.Error().Msgf("ReadLinesFromFile failed: %v", err)
		return nil, err
	}
	defer file.Close()

	// create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// read and print each line of the file
	for scanner.Scan() {
		line := scanner.Text()

		// in case this is the bank of words we fill, make sure each entry is valid
		// a valid word is at least 3 words, and is alphabetic
		if isBank {
			if len(line) >= 3 && alphaRegexp.MatchString(line) {
				bankOfWords[line] = struct{}{}
			}
		} else {
			lines = append(lines, line)
		}
	}

	// check for any errors during the file read operation
	if err := scanner.Err(); err != nil {
		log.Error().Msgf("scanner.Err()—: %v", err)
		return nil, err
	}
	return lines, nil
}

// FindArticleText extract's the article's body of the html resource
func FindArticleText(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "div" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "article-text") {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := FindArticleText(c); result != nil {
			return result
		}
	}
	return nil
}

// ExtractText loops through each html.node and returns the text
func ExtractText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += ExtractText(c)
	}
	return result
}

// RandomString generates a random string in a size of given length
func RandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length+2)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}

// MakeTmpUnderPWD get PWD and makes /tmp under it
func MakeTmpUnderPWD(tmp string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal().Msgf("os.Getwd() failed: %v", err)
		return "", err
	}
	//make tmp dir
	if err := os.Mkdir(pwd+tmp, 0750); err != nil && !os.IsExist(err) {
		log.Fatal().Msgf("Mkdir failed: %v", err)
		return "", err
	}

	return pwd + tmp, nil
}

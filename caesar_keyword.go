package main

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

func getAlphabet(keyword string, shift int) (string, error) {
	if !(0 <= shift && shift < 26) {
		return "", errors.New("shift must be between 0 and 26")
	}

	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	ret := make([]rune, len(alphabet))

	idx := shift
	nextIdx := func(idx int) int {
		idx++
		if idx == len(alphabet) {
			idx = 0
		}
		return idx
	}

	visited := make(map[rune]struct{}, len(alphabet))
	for _, char := range keyword {
		if !unicode.IsLetter(char) {
			continue
		}
		char = unicode.ToUpper(char)
		if _, ok := visited[char]; !ok {
			visited[char] = struct{}{}
			ret[idx] = char
			idx = nextIdx(idx)
		}
	}

	alphabetIdx := 0
	for len(visited) < len(alphabet) {
		char := rune(alphabet[alphabetIdx])
		if _, ok := visited[char]; !ok {
			visited[char] = struct{}{}
			ret[idx] = char
			idx = nextIdx(idx)
		}
		alphabetIdx = nextIdx(alphabetIdx)
	}

	return string(ret), nil
}

func Encrypt(text, keyword string, shift int) (string, error) {
	alphabet, err := getAlphabet(keyword, shift)
	if err != nil {
		return "", fmt.Errorf("failed to get alphabet: %w", err)
	}

	ret := strings.Builder{}
	for _, char := range text {
		newChar := char
		if unicode.IsLetter(newChar) {
			newChar = rune(alphabet[unicode.ToUpper(newChar)-'A'])
			if unicode.IsLower(char) {
				newChar = unicode.ToLower(newChar)
			}
		}
		ret.WriteRune(newChar)
	}

	return ret.String(), nil
}

func Decrypt(text, keyword string, shift int) (string, error) {
	alphabet, err := getAlphabet(keyword, shift)
	if err != nil {
		return "", fmt.Errorf("failed to get alphabet: %w", err)
	}

	mapping := make(map[rune]int, len(alphabet))
	for i, char := range alphabet {
		mapping[char] = i
	}

	ret := strings.Builder{}
	for _, char := range text {
		newChar := char
		if unicode.IsLetter(newChar) {
			newChar = rune(mapping[unicode.ToUpper(newChar)] + 'A')
			if unicode.IsLower(char) {
				newChar = unicode.ToLower(newChar)
			}
		}
		ret.WriteRune(newChar)
	}

	return ret.String(), nil
}

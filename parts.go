package syslogsidecar

import (
	"fmt"
)

//
// Subset of https://github.com/linkdotnet/golang-stringbuilder/blob/main/stringbuilder.go
//

type parts struct {
	data     []rune
	position int
}

// Creates a new instance of the parts with preallocated array
func newparts(initialCapacity int) *parts {
	return &parts{data: make([]rune, initialCapacity)}
}

// Appends a text to the parts instance
func (p *parts) appendText(text string) int {
	if len(text) == 0 {
		return 0
	}

	p.resize(text)
	textRunes := []rune(text)
	copy(p.data[p.position:], textRunes)
	l := len(textRunes)
	p.position = p.position + l

	return l
}

// Appends a single character to the parts instance
func (p *parts) appendRune(char rune) int {
	newLen := p.position + 1
	if newLen >= cap(p.data) {
		p.grow(newLen)
	}
	p.data[p.position] = char
	p.position++

	return 1
}

// Returns the current position
func (p *parts) pos() int {
	return p.position
}

// Sets the position to 0.
// The internal array will stay the same.
func (p *parts) rewind() {
	p.position = 0
}

// Change current position
func (p *parts) skip(forward int) error {
	if forward <= 0 {
		return fmt.Errorf("forward should always be greater than zero")
	}

	newPos := p.position + forward

	if newPos >= len(p.data) {
		return fmt.Errorf("cannot skip after end")
	}

	p.position = newPos

	return nil
}

// Gets the rune at the specific index
func (p *parts) runeAt(index int) (rune, error) {
	if index < 0 {
		return 0, fmt.Errorf("index should always be greater than or equal to zero")
	}
	if index >= len(p.data) {
		return 0, fmt.Errorf("index cannot be greater than current position")
	}
	return p.data[index], nil
}

// Sets the rune at the specific position
func (p *parts) setRuneAt(index int, val rune) error {
	if index < 0 {
		return fmt.Errorf("index should always be greater than or equal to zero")
	}
	if index >= len(p.data) {
		return fmt.Errorf("invalid index")
	}
	p.data[index] = val

	return nil
}

func (p *parts) resize(text string) {
	newLen := p.position + len(text)
	if newLen > cap(p.data) {
		p.grow(newLen)
	}
}

func (p *parts) grow(lenToAdd int) {
	// Grow times 2 until lenToAdd fits
	newLen := len(p.data)

	if newLen == 0 {
		newLen = 8
	}

	for newLen < lenToAdd {
		newLen = newLen * 2
	}

	p.data = append(p.data, make([]rune, newLen-len(p.data))...)
}

func (p *parts) part(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}

	start := p.position

	err := p.skip(length)
	if err != nil {
		return "", err
	}

	end := p.position

	r := make([]rune, end-start)
	copy(r, p.data[start:end])

	return string(r), nil
}

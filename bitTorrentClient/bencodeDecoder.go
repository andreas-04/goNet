/*
    Integers are encoded by preceding the number in base 10 ASCII format with i and
	ending it with e. i<integer encoded in base ten ASCII>e. E.g. 11 is encoded as i11e

    Byte strings are preceded with the length of the byte string in base 10 and a :.Byte
	strings aren’t limited to visible ASCII characters. <length in base 10>:<byte string>.
	E.g. helicopter is encoded as 10:helicopter

    Lists are encoded by preceding the concatenation of all values in the list with l
	(the character l) and postfixing it with e. l<contents>e. E.g. a list of “helicopter”
	and the number 11 would be li11e10:helicoptere

    Dicts are encoded by preceding the concatenation of all key-value pairs with d and
	postfixing it with e. Keys can only be byte strings and the pair is encoded by
	concatenating the key and value without any delimiters. d<<k1><v1><k2><v2>…>e
*/

package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

func decodeInt(data *bytes.Reader) (int, error) {
	ch, _ := data.ReadByte()
	if ch != 'i' {
		return 0, fmt.Errorf("expected 'i', got %c", ch)
	}
	var numStr []byte
	for {
		ch, err := data.ReadByte()
		if err != nil {
			return 0, err
		}
		if ch == 'e' {
			break
		}
		numStr = append(numStr, ch)
	}
	return strconv.Atoi(string(numStr))
}

func decodeString(data *bytes.Reader) (string, error) {
	var lengthStr []byte
	for {
		ch, _ := data.ReadByte()
		if ch == ':' {
			break
		}
		lengthStr = append(lengthStr, ch)
	}
	length, err := strconv.Atoi(string(lengthStr))
	if err != nil {
		return "", err
	}
	strBytes := make([]byte, length)
	_, err = io.ReadFull(data, strBytes)
	return string(strBytes), err
}

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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

type BencodeDecoder struct {
	reader *bufio.Reader
}

func NewDecoder(r io.Reader) *BencodeDecoder {
	return &BencodeDecoder{reader: bufio.NewReader(r)}
}

func (d *BencodeDecoder) next() (byte, error) {
	return d.reader.ReadByte()
}

func (d *BencodeDecoder) peek() (byte, error) {
	ch, err := d.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	_ = d.reader.UnreadByte() // Push it back
	return ch, nil
}

func (d *BencodeDecoder) decode() (interface{}, error) {
	ch, err := d.peek()
	if err != nil {
		return nil, err
	}

	switch {
	case ch == 'i': // Integer
		return d.decodeInt()
	case ch >= '0' && ch <= '9': // String
		return d.decodeString()
	case ch == 'l': // List
		return d.decodeList()
	case ch == 'd': // Dictionary
		return d.decodeDict()
	default:
		return nil, fmt.Errorf("unexpected character '%c'", ch)
	}
}

// Decode an integer (i<num>e)
func (d *BencodeDecoder) decodeInt() (int, error) {
	_, err := d.next()
	if err != nil {
		return 0, err
	}

	var numStr []byte
	for {
		ch, err := d.next()
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

// Decode a string (length:string)
func (d *BencodeDecoder) decodeString() (string, error) {
	var lengthStr []byte

	for {
		ch, err := d.next()
		if err != nil {
			return "", err
		}
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
	_, err = io.ReadFull(d.reader, strBytes)
	return string(strBytes), err
}

// Decode a list (l...e)
func (d *BencodeDecoder) decodeList() ([]interface{}, error) {
	_, err := d.next() // Consume 'l'
	if err != nil {
		return nil, err
	}

	var list []interface{}
	for {
		ch, err := d.peek()
		if err != nil {
			return nil, err
		}
		if ch == 'e' {
			_, _ = d.next()
			break
		}

		item, err := d.decode()
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}

	return list, nil
}

// Decode a dictionary (d...e)
func (d *BencodeDecoder) decodeDict() (map[string]interface{}, error) {
	_, err := d.next()
	if err != nil {
		return nil, err
	}

	dict := make(map[string]interface{})
	for {
		ch, err := d.peek()
		if err != nil {
			return nil, err
		}
		if ch == 'e' {
			_, _ = d.next()
			break
		}

		key, err := d.decodeString()
		if err != nil {
			return nil, err
		}

		value, err := d.decode()
		if err != nil {
			return nil, err
		}

		dict[key] = value
	}

	return dict, nil
}

func writeToFile(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	_, err = file.Write(jsonData)
	return err
}

func main() {
	file, err := os.Open("example.torrent")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	decoder := NewDecoder(file)

	data, err := decoder.decode()
	if err != nil {
		fmt.Println("Error decoding file:", err)
		return
	}

	outputFilename := "decoded_output.json"
	err = writeToFile(data, outputFilename)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("Decoded data written to", outputFilename)
}

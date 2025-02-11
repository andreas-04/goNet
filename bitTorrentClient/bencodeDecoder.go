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
package bittorrentclient

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type Torrent struct {
	Announce     string
	AnnounceList [][]string
	CreationDate int64
	Comment      string
	CreatedBy    string
	Info         TorrentInfo
}

type TorrentInfo struct {
	PieceLength int64
	Pieces      []byte
	Private     int64
	Name        string
	Length      int64
	Files       []TorrentFile
}

type TorrentFile struct {
	Length int64
	Path   []string
}

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
	_ = d.reader.UnreadByte()
	return ch, nil
}

func (d *BencodeDecoder) decode() (interface{}, error) {
	ch, err := d.peek()
	if err != nil {
		return nil, err
	}

	switch {
	case ch == 'i':
		return d.decodeInt()
	case ch >= '0' && ch <= '9':
		return d.decodeString()
	case ch == 'l':
		return d.decodeList()
	case ch == 'd':
		return d.decodeDict()
	default:
		return nil, fmt.Errorf("unexpected character '%c'", ch)
	}
}

func (d *BencodeDecoder) decodeInt() (int64, error) {
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

	return strconv.ParseInt(string(numStr), 10, 64)
}

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

	length, err := strconv.ParseInt(string(lengthStr), 10, 64)
	if err != nil {
		return "", err
	}

	strBytes := make([]byte, length)
	_, err = io.ReadFull(d.reader, strBytes)
	return string(strBytes), err
}

func (d *BencodeDecoder) decodeList() ([]interface{}, error) {
	_, err := d.next()
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

func DecodeTorrent(r io.Reader) (*Torrent, error) {
	decoder := NewDecoder(r)
	data, err := decoder.decode()
	if err != nil {
		return nil, err
	}
	return parseTorrent(data)
}

func parseTorrent(data interface{}) (*Torrent, error) {
	topLevel, ok := data.(map[string]interface{})
	if !ok {
		return nil, errors.New("top-level data is not a dictionary")
	}

	torrent := &Torrent{}

	if announce, ok := topLevel["announce"].(string); ok {
		torrent.Announce = announce
	} else {
		return nil, errors.New("missing required field 'announce'")
	}

	if announceListInterface, ok := topLevel["announce-list"]; ok {
		announceList, ok := announceListInterface.([]interface{})
		if !ok {
			return nil, errors.New("announce-list is not a list")
		}
		for _, tierInterface := range announceList {
			tier, ok := tierInterface.([]interface{})
			if !ok {
				return nil, errors.New("announce-list tier is not a list")
			}
			var tierUrls []string
			for _, urlInterface := range tier {
				url, ok := urlInterface.(string)
				if !ok {
					return nil, errors.New("announce-list contains non-string URL")
				}
				tierUrls = append(tierUrls, url)
			}
			torrent.AnnounceList = append(torrent.AnnounceList, tierUrls)
		}
	}

	if creationDateInterface, ok := topLevel["creation date"]; ok {
		creationDate, ok := creationDateInterface.(int64)
		if !ok {
			return nil, errors.New("creation date is not an integer")
		}
		torrent.CreationDate = creationDate
	}

	if comment, ok := topLevel["comment"].(string); ok {
		torrent.Comment = comment
	}

	if createdBy, ok := topLevel["created by"].(string); ok {
		torrent.CreatedBy = createdBy
	}

	infoInterface, ok := topLevel["info"]
	if !ok {
		return nil, errors.New("missing required field 'info'")
	}
	infoMap, ok := infoInterface.(map[string]interface{})
	if !ok {
		return nil, errors.New("info is not a dictionary")
	}
	info, err := parseTorrentInfo(infoMap)
	if err != nil {
		return nil, fmt.Errorf("error parsing info: %v", err)
	}
	torrent.Info = *info

	return torrent, nil
}

func parseTorrentInfo(infoMap map[string]interface{}) (*TorrentInfo, error) {
	info := &TorrentInfo{}

	if pieceLengthInterface, ok := infoMap["piece length"]; ok {
		pieceLength, ok := pieceLengthInterface.(int64)
		if !ok {
			return nil, errors.New("piece length is not an integer")
		}
		info.PieceLength = pieceLength
	} else {
		return nil, errors.New("missing required field 'piece length'")
	}

	if piecesInterface, ok := infoMap["pieces"]; ok {
		pieces, ok := piecesInterface.(string)
		if !ok {
			return nil, errors.New("pieces is not a string")
		}
		info.Pieces = []byte(pieces)
	} else {
		return nil, errors.New("missing required field 'pieces'")
	}

	if name, ok := infoMap["name"].(string); ok {
		info.Name = name
	} else {
		return nil, errors.New("missing required field 'name'")
	}

	if privateInterface, ok := infoMap["private"]; ok {
		private, ok := privateInterface.(int64)
		if !ok {
			return nil, errors.New("private is not an integer")
		}
		info.Private = private
	}

	_, hasLength := infoMap["length"]
	_, hasFiles := infoMap["files"]

	if hasLength && hasFiles {
		return nil, errors.New("info contains both length and files")
	}

	if hasLength {
		length, ok := infoMap["length"].(int64)
		if !ok {
			return nil, errors.New("length is not an integer")
		}
		info.Length = length
	} else if hasFiles {
		filesInterface, _ := infoMap["files"]
		filesList, ok := filesInterface.([]interface{})
		if !ok {
			return nil, errors.New("files is not a list")
		}
		for _, fileInterface := range filesList {
			fileMap, ok := fileInterface.(map[string]interface{})
			if !ok {
				return nil, errors.New("file entry is not a dictionary")
			}
			file := TorrentFile{}
			lengthInterface, ok := fileMap["length"]
			if !ok {
				return nil, errors.New("file missing length")
			}
			length, ok := lengthInterface.(int64)
			if !ok {
				return nil, errors.New("file length is not an integer")
			}
			file.Length = length
			pathInterface, ok := fileMap["path"]
			if !ok {
				return nil, errors.New("file missing path")
			}
			pathList, ok := pathInterface.([]interface{})
			if !ok {
				return nil, errors.New("file path is not a list")
			}
			var path []string
			for _, p := range pathList {
				pathPart, ok := p.(string)
				if !ok {
					return nil, errors.New("path part is not a string")
				}
				path = append(path, pathPart)
			}
			file.Path = path
			info.Files = append(info.Files, file)
		}
	} else {
		return nil, errors.New("info missing both length and files")
	}

	return info, nil
}

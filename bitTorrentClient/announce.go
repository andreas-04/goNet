// This file handles announcing we are joining the swarm and returns an announcer type
package bittorrentclient

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/jackpal/bencode-go"
)

type Announcer struct {
	announce_url string
	piece_size   int64
	TotalSize    int64
	urlParams    urlParams
}

type urlParams struct {
	info_dict  string
	peer_id    string
	port       string
	uploaded   int64
	downloaded int64
	left       int64
	compact    string
	event      string
}

func NewAnnouncer(filepath string) *Announcer {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()
	var torrent map[string]interface{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&torrent)
	if err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}

	// Extract the "info" dictionary
	info, ok := torrent["info"].(map[string]interface{})
	if !ok {
		log.Fatalf("Error: 'info' field is missing or not a dictionary")
	}

	//  Bencode the info-dict
	var bencodedInfo bytes.Buffer
	err = bencode.Marshal(&bencodedInfo, info)
	if err != nil {
		log.Fatalf("Error bencoding info: %v", err)
	}

	sha1_info_dict := computeInfoHash(bencodedInfo.Bytes())
	uniquePeerId := generatePeerId()
	url := torrent["announce"].(string)
	piece_len := torrent["piece length"].(int64)
	length, err := GetTotalLength(torrent)
	if err != nil {
		panic(err)
	}
	return &Announcer{
		announce_url: url,
		piece_size:   piece_len,
		urlParams: urlParams{
			info_dict:  sha1_info_dict,
			peer_id:    uniquePeerId,
			port:       "6881",
			uploaded:   0,
			downloaded: 0,
			left:       length,
			compact:    "1",
			event:      "started",
		},
	}
}

// this function returns the hased sha-1 string of the info-dict
func computeInfoHash(bencodedInfo []byte) string {
	hash := sha1.Sum(bencodedInfo)
	return url.QueryEscape(string(hash[:]))
}

func generatePeerId() string {
	buf := make([]byte, 20)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return url.QueryEscape(string(buf))
}

// this function calculates the total size of all files in the torrent
func GetTotalLength(decoded map[string]interface{}) (int64, error) {
	info, ok := decoded["info"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("missing 'info' dictionary")
	}

	// Check for single-file torrent
	if length, ok := info["length"].(int64); ok {
		return length, nil
	}

	// Multi-file torrent
	files, ok := info["files"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("missing 'files' list in multi-file torrent")
	}

	var total int64
	for _, f := range files {
		file, ok := f.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("invalid file entry format")
		}

		length, ok := file["length"].(int64)
		if !ok {
			return 0, fmt.Errorf("file entry missing length")
		}

		total += length
	}

	return total, nil
}

// this function generates a request url
func (a Announcer) generateEncodedURL() string {
	params := url.Values{}
	params.Set("info_hash", a.urlParams.info_dict)
	params.Set("peer_id", a.urlParams.peer_id)
	params.Set("port", a.urlParams.port)
	params.Set("uploaded", strconv.FormatInt(a.urlParams.uploaded, 10))
	params.Set("downloaded", strconv.FormatInt(a.urlParams.downloaded, 10))
	params.Set("left", strconv.FormatInt(a.urlParams.left, 10))
	params.Set("compact", a.urlParams.compact)
	if a.urlParams.event != "" {
		params.Set("event", a.urlParams.event)
	}
	encoded_params := params.Encode()
	return a.announce_url + "?" + encoded_params
}

// this function is to be called whenever a new piece is recieve, it will update the downloaded, and left url params
func (a *Announcer) handleNewPieceLeeched(bytesDownloaded int64) {
	a.urlParams.downloaded += bytesDownloaded
	a.urlParams.left = a.TotalSize - a.urlParams.downloaded
}

// function to update the uploaded url param whenever a new piece is seeded
func (a *Announcer) handleNewPieceSeeded() {
	a.urlParams.uploaded += a.piece_size
}

// function used to update the event in the Announcer
func (a *Announcer) setEvent(newEvent string) {
	switch newEvent {
	case "started":
		a.urlParams.event = newEvent
	case "completed":
		a.urlParams.event = newEvent
	case "stopped":
		a.urlParams.event = newEvent
	default:
		a.urlParams.event = ""
	}
}

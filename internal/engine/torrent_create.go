package engine

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CreatedTorrent contains information about the generated torrent file.
type CreatedTorrent struct {
	TorrentPath string
	InfoHash    string
	Magnet      string
	TotalBytes  int64
	PieceLength int64
	PiecesCount int
}

// TorrentCreateOptions specifies options for generating a .torrent file.
type TorrentCreateOptions struct {
	Comment       string
	Trackers      []string
	PieceLength   int64 // 0 for auto
	OutPath       string
}

// CreateTorrent turns a local file or directory into a .torrent file and returns its metadata.
func CreateTorrent(sourcePath string, outPath string, opts *TorrentCreateOptions) (*CreatedTorrent, error) {
	if opts == nil {
		opts = &TorrentCreateOptions{}
	}

	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat source: %w", err)
	}

	trackers := opts.Trackers
	if len(trackers) == 0 {
		trackers = DefaultPublicTrackers
	}

	var totalBytes int64
	var fileEntries []fileMeta

	if fi.IsDir() {
		err = filepath.Walk(absPath, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(absPath, path)
			if relErr != nil {
				return relErr
			}
			// Skip .torrent files or hidden files like .DS_Store
			if strings.HasSuffix(rel, ".torrent") || filepath.Base(rel) == ".DS_Store" {
				return nil
			}
			parts := strings.Split(filepath.ToSlash(rel), "/")
			fileEntries = append(fileEntries, fileMeta{
				path:      path,
				relParts:  parts,
				sizeBytes: info.Size(),
			})
			totalBytes += info.Size()
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk directory: %w", err)
		}
		if len(fileEntries) == 0 {
			return nil, fmt.Errorf("directory %q contains no files to seed", sourcePath)
		}
		// Sort files by path for deterministic bencode
		sort.Slice(fileEntries, func(i, j int) bool {
			return strings.Join(fileEntries[i].relParts, "/") < strings.Join(fileEntries[j].relParts, "/")
		})
	} else {
		totalBytes = fi.Size()
		fileEntries = append(fileEntries, fileMeta{
			path:      absPath,
			relParts:  []string{fi.Name()},
			sizeBytes: fi.Size(),
		})
	}

	// Calculate optimal piece length if not specified
	pieceLength := opts.PieceLength
	if pieceLength <= 0 {
		pieceLength = calculateOptimalPieceLength(totalBytes)
	}

	// Compute piece hashes
	piecesConcat, pieceCount, err := hashPieces(fileEntries, pieceLength)
	if err != nil {
		return nil, fmt.Errorf("hash pieces: %w", err)
	}

	// Build bencode Info Dictionary
	infoDict := make(map[string]interface{})
	infoDict["name"] = fi.Name()
	infoDict["piece length"] = pieceLength
	infoDict["pieces"] = piecesConcat

	if fi.IsDir() {
		var filesList []interface{}
		for _, f := range fileEntries {
			fDict := make(map[string]interface{})
			fDict["length"] = f.sizeBytes
			var pathParts []interface{}
			for _, p := range f.relParts {
				pathParts = append(pathParts, p)
			}
			fDict["path"] = pathParts
			filesList = append(filesList, fDict)
		}
		infoDict["files"] = filesList
	} else {
		infoDict["length"] = totalBytes
	}

	// Bencode Info Dictionary to calculate InfoHash
	infoBencoded := bencodeDict(infoDict)
	h := sha1.New()
	h.Write(infoBencoded)
	infoHashBytes := h.Sum(nil)
	infoHashHex := hex.EncodeToString(infoHashBytes)

	// Build root dictionary
	rootDict := make(map[string]interface{})
	rootDict["announce"] = trackers[0]

	var announceList []interface{}
	for _, tr := range trackers {
		announceList = append(announceList, []interface{}{tr})
	}
	rootDict["announce-list"] = announceList

	rootDict["created by"] = "anpan"
	rootDict["creation date"] = time.Now().Unix()
	if opts.Comment != "" {
		rootDict["comment"] = opts.Comment
	}
	rootDict["info"] = infoDict

	rootBencoded := bencodeDict(rootDict)

	// Target torrent path
	destTorrentPath := outPath
	if destTorrentPath == "" {
		destTorrentPath = absPath + ".torrent"
	}

	if err := os.WriteFile(destTorrentPath, rootBencoded, 0644); err != nil {
		return nil, fmt.Errorf("write torrent file: %w", err)
	}

	magnet := BuildMagnet(infoHashHex, fi.Name(), trackers)

	return &CreatedTorrent{
		TorrentPath: destTorrentPath,
		InfoHash:    infoHashHex,
		Magnet:      magnet,
		TotalBytes:  totalBytes,
		PieceLength: pieceLength,
		PiecesCount: pieceCount,
	}, nil
}

type fileMeta struct {
	path      string
	relParts  []string
	sizeBytes int64
}

func calculateOptimalPieceLength(totalBytes int64) int64 {
	switch {
	case totalBytes < 50*1024*1024:
		return 256 * 1024 // 256 KB
	case totalBytes < 512*1024*1024:
		return 512 * 1024 // 512 KB
	case totalBytes < 1024*1024*1024:
		return 1024 * 1024 // 1 MB
	case totalBytes < 4*1024*1024*1024:
		return 2048 * 1024 // 2 MB
	default:
		return 4096 * 1024 // 4 MB
	}
}

func hashPieces(files []fileMeta, pieceLength int64) ([]byte, int, error) {
	var piecesBuf bytes.Buffer
	h := sha1.New()
	buf := make([]byte, 64*1024)
	var currentPieceWritten int64
	pieceCount := 0

	for _, f := range files {
		file, err := os.Open(f.path)
		if err != nil {
			return nil, 0, err
		}

		for {
			toRead := int64(len(buf))
			needed := pieceLength - currentPieceWritten
			if needed < toRead {
				toRead = needed
			}

			n, rErr := file.Read(buf[:toRead])
			if n > 0 {
				h.Write(buf[:n])
				currentPieceWritten += int64(n)

				if currentPieceWritten == pieceLength {
					piecesBuf.Write(h.Sum(nil))
					h.Reset()
					currentPieceWritten = 0
					pieceCount++
				}
			}
			if rErr != nil {
				if rErr == io.EOF {
					break
				}
				file.Close()
				return nil, 0, rErr
			}
		}
		file.Close()
	}

	// Final partial piece
	if currentPieceWritten > 0 {
		piecesBuf.Write(h.Sum(nil))
		pieceCount++
	}

	return piecesBuf.Bytes(), pieceCount, nil
}

// Simple deterministic Bencode encoder
func bencodeDict(dict map[string]interface{}) []byte {
	var buf bytes.Buffer
	buf.WriteByte('d')

	var keys []string
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		bencodeString(&buf, k)
		bencodeValue(&buf, dict[k])
	}

	buf.WriteByte('e')
	return buf.Bytes()
}

func bencodeValue(buf *bytes.Buffer, v interface{}) {
	switch val := v.(type) {
	case string:
		bencodeString(buf, val)
	case []byte:
		buf.WriteString(fmt.Sprintf("%d:", len(val)))
		buf.Write(val)
	case int:
		buf.WriteString(fmt.Sprintf("i%de", val))
	case int64:
		buf.WriteString(fmt.Sprintf("i%de", val))
	case []interface{}:
		buf.WriteByte('l')
		for _, item := range val {
			bencodeValue(buf, item)
		}
		buf.WriteByte('e')
	case map[string]interface{}:
		buf.Write(bencodeDict(val))
	}
}

func bencodeString(buf *bytes.Buffer, s string) {
	buf.WriteString(fmt.Sprintf("%d:%s", len(s), s))
}

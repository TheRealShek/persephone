package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

/*
ReadIndex reads and deserializes the .purr/index file.
Git index format: 12-byte header + repeated entries.
Each entry: fixed metadata (62 bytes) + variable-length path + padding.
*/
func ReadIndex(indexPath string) ([]IndexEntry, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	buf := bytes.NewReader(data) // creates a readable buffer from a byte slice
	var entries []IndexEntry

	// Read and validate 12-byte header
	header := make([]byte, 12)
	if n, err := buf.Read(header); err != nil || n != 12 {
		return nil, fmt.Errorf("failed to read index header: %w", err)
	}
	if string(header[:4]) != "DIRC" {
		return nil, fmt.Errorf("invalid index header: expected DIRC, got %s", string(header[:4]))
	}

	for buf.Len() > 0 {
		var entry IndexEntry

		// Reading fixed 62-byte metadata block from the memory buffer (buf) and storing its value into different variables
		// Read IndexEntry struct to better understand it
		// binary.BigEndian defines byte order: most significant byte comes first

		var ctime, mtime int64
		if err := binary.Read(buf, binary.BigEndian, &ctime); err != nil {
			return nil, fmt.Errorf("failed to read ctime: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &mtime); err != nil {
			return nil, fmt.Errorf("failed to read mtime: %w", err)
		}
		entry.Ctime = time.Unix(ctime, 0)
		entry.Mtime = time.Unix(mtime, 0)

		if err := binary.Read(buf, binary.BigEndian, &entry.Dev); err != nil {
			return nil, fmt.Errorf("failed to read dev: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &entry.Ino); err != nil {
			return nil, fmt.Errorf("failed to read ino: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &entry.Mode); err != nil {
			return nil, fmt.Errorf("failed to read mode: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &entry.Uid); err != nil {
			return nil, fmt.Errorf("failed to read uid: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &entry.Gid); err != nil {
			return nil, fmt.Errorf("failed to read gid: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &entry.Size); err != nil {
			return nil, fmt.Errorf("failed to read size: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &entry.Sha1); err != nil {
			return nil, fmt.Errorf("failed to read sha1: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &entry.Stage); err != nil {
			return nil, fmt.Errorf("failed to read stage: %w", err)
		}

		/*
			The padding calculation and subsequent `buf.Seek` ensures that each IndexEntry starts at an 8-byte aligned offset in the index file, as required by the Purr index format.
		*/
		var pathLen uint16
		if err := binary.Read(buf, binary.BigEndian, &pathLen); err != nil {
			return nil, fmt.Errorf("failed to read path length: %w", err)
		}

		pathBytes := make([]byte, pathLen)
		if n, err := buf.Read(pathBytes); err != nil || n != int(pathLen) {
			return nil, fmt.Errorf("failed to read path (expected %d bytes): %w", pathLen, err)
		}
		entry.Path = string(pathBytes)

		// Skip padding to align next entry to 8 bytes
		// Total entry size = 62 (metadata) + 2 (path length) + pathLen (path data)
		entrySize := 62 + 2 + pathLen
		paddingLen := (8 - (entrySize % 8)) % 8
		if _, err := buf.Seek(int64(paddingLen), io.SeekCurrent); err != nil {
			return nil, fmt.Errorf("failed to skip padding: %w", err)
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

// WriteIndex serializes index entries and writes them to disk in Git index format.
// Format: 12-byte header (DIRC + version 2 + entry count) + entries with 8-byte padding.
func WriteIndex(indexPath string, entries []IndexEntry) error {
	var buf bytes.Buffer

	// Write 12-byte header
	buf.WriteString("DIRC")                                    // Magic signature (4 bytes)
	binary.Write(&buf, binary.BigEndian, uint32(2))            // Version 2 (4 bytes)
	binary.Write(&buf, binary.BigEndian, uint32(len(entries))) // Entry count (4 bytes)

	// Write each entry
	for _, entry := range entries {
		// Write fixed 62-byte metadata
		binary.Write(&buf, binary.BigEndian, entry.Ctime.Unix()) // 8 bytes
		binary.Write(&buf, binary.BigEndian, entry.Mtime.Unix()) // 8 bytes
		binary.Write(&buf, binary.BigEndian, entry.Dev)          // 4 bytes
		binary.Write(&buf, binary.BigEndian, entry.Ino)          // 4 bytes
		binary.Write(&buf, binary.BigEndian, entry.Mode)         // 4 bytes
		binary.Write(&buf, binary.BigEndian, entry.Uid)          // 4 bytes
		binary.Write(&buf, binary.BigEndian, entry.Gid)          // 4 bytes
		binary.Write(&buf, binary.BigEndian, entry.Size)         // 4 bytes
		binary.Write(&buf, binary.BigEndian, entry.Sha1)         // 20 bytes
		binary.Write(&buf, binary.BigEndian, entry.Stage)        // 2 bytes

		// Write path length and path data
		pathBytes := []byte(entry.Path)
		binary.Write(&buf, binary.BigEndian, uint16(len(pathBytes))) // 2 bytes
		buf.Write(pathBytes)

		// Add padding to align to 8-byte boundary
		entrySize := 62 + 2 + len(pathBytes)
		paddingLen := (8 - (entrySize % 8)) % 8
		buf.Write(make([]byte, paddingLen))
	}

	// Write to disk
	if err := os.WriteFile(indexPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

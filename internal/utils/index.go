package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// ReadIndex deserializes the `.purr/index` file.
// The index format is identical to Git's index version 2, designed for fast random access and portability:
//  1. A 12-byte header:
//     - 4 bytes signature: "DIRC" (Directory Cache)
//     - 4 bytes version: big-endian uint32 (2)
//     - 4 bytes entry count: big-endian uint32 (number of files staged)
//  2. Staged entries laid out sequentially. Each entry contains:
//     - 62 bytes of fixed-length metadata (stat cache fields)
//     - 2 bytes path length: big-endian uint16
//     - Variable-length path byte slice
//     - 0 to 7 bytes of null-byte padding to align the start of the next entry to an 8-byte boundary.
func ReadIndex(indexPath string) ([]IndexEntry, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	buf := bytes.NewReader(data)
	var entries []IndexEntry

	// Read and validate 12-byte header
	header := make([]byte, 12)
	if n, err := buf.Read(header); err != nil || n != 12 {
		return nil, fmt.Errorf("failed to read index header: %w", err)
	}
	if string(header[:4]) != "DIRC" {
		return nil, fmt.Errorf("invalid index header: expected DIRC, got %s", string(header[:4]))
	}

	version := binary.BigEndian.Uint32(header[4:8])
	if version != 2 {
		return nil, fmt.Errorf("unsupported index version: expected 2, got %d", version)
	}

	entryCount := binary.BigEndian.Uint32(header[8:12])

	for i := uint32(0); i < entryCount; i++ {
		var entry IndexEntry

		// Read the 62-byte fixed-size metadata block.
		// Layout details:
		//  - Ctime/Mtime: 8 bytes each (Unix epoch seconds)
		//  - Dev/Ino/Mode/Uid/Gid/Size: 4 bytes each
		//  - Sha1: 20 bytes
		//  - Stage flags: 2 bytes
		// We use BigEndian serialization to guarantee platform-independent repository portability.
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

		// Read variable length path
		var pathLen uint16
		if err := binary.Read(buf, binary.BigEndian, &pathLen); err != nil {
			return nil, fmt.Errorf("failed to read path length: %w", err)
		}

		pathBytes := make([]byte, pathLen)
		if n, err := buf.Read(pathBytes); err != nil || n != int(pathLen) {
			return nil, fmt.Errorf("failed to read path (expected %d bytes): %w", pathLen, err)
		}
		entry.Path = string(pathBytes)

		// 8-byte alignment logic:
		// Index formats require entries to be aligned to 8-byte boundary offsets relative
		// to the file start. We compute the exact written bytes for the current entry
		// (62 bytes fixed metadata + 2 bytes path size field + path data length) and
		// skip the remaining padding bytes to align the read pointer for the next entry.
		entrySize := 62 + 2 + pathLen
		paddingLen := (8 - (entrySize % 8)) % 8
		if _, err := buf.Seek(int64(paddingLen), io.SeekCurrent); err != nil {
			return nil, fmt.Errorf("failed to skip padding: %w", err)
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

// WriteIndex serializes the staged index entries back to `.purr/index`.
// It maintains deterministic sorting and layout standards, ensuring that equivalent working states
// generate byte-for-byte identical index files.
func WriteIndex(indexPath string, entries []IndexEntry) error {
	var buf bytes.Buffer

	// Write 12-byte header
	buf.WriteString("DIRC")                                    // Magic signature (4 bytes)
	if err := binary.Write(&buf, binary.BigEndian, uint32(2)); err != nil {
		return err
	}
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(entries))); err != nil {
		return err
	}

	// Write each entry sequentially
	for _, entry := range entries {
		// Write fixed 62-byte metadata block
		if err := binary.Write(&buf, binary.BigEndian, entry.Ctime.Unix()); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Mtime.Unix()); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Dev); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Ino); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Mode); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Uid); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Gid); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Size); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Sha1); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.BigEndian, entry.Stage); err != nil {
			return err
		}

		// Write path length and data
		pathBytes := []byte(entry.Path)
		if err := binary.Write(&buf, binary.BigEndian, uint16(len(pathBytes))); err != nil {
			return err
		}
		buf.Write(pathBytes)

		// Compute and write null-byte padding to satisfy the 8-byte alignment constraint
		entrySize := 62 + 2 + len(pathBytes)
		paddingLen := (8 - (entrySize % 8)) % 8
		buf.Write(make([]byte, paddingLen))
	}

	// Write to disk atomically (0644 standard file permissions)
	if err := os.WriteFile(indexPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}


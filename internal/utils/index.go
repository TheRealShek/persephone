package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

// ReadIndex deserializes the `.purr/index` file.
// The index format is based on Git's index, designed for fast random access and portability:
//  1. A 12-byte header:
//     - 4 bytes signature: "DIRC" (Directory Cache)
//     - 4 bytes version: big-endian uint32 (2 or 3)
//     - 4 bytes entry count: big-endian uint32 (number of files staged)
//  2. Staged entries laid out sequentially. Each entry contains:
//     - 64 bytes of fixed-length metadata (stat cache fields + path length)
//     - Variable-length path byte slice
//     - Padding: version 2 uses 0-7 NUL bytes, version 3 uses 1-8 NUL bytes (Git spec compliant).
//
// Version 3 stores timestamps as {uint32 seconds, uint32 nanoseconds} pairs to preserve
// nanosecond precision for stat-cache comparison, fixing the mtime false-mismatch bug in version 2
// where nanoseconds were discarded.
func ReadIndex(indexPath string) ([]IndexEntry, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("index file too short for header")
	}

	// Validate 12-byte header
	if string(data[:4]) != "DIRC" {
		return nil, fmt.Errorf("invalid index header: expected DIRC, got %s", string(data[:4]))
	}

	version := binary.BigEndian.Uint32(data[4:8])
	if version != 2 && version != 3 {
		return nil, fmt.Errorf("unsupported index version: expected 2 or 3, got %d", version)
	}

	entryCount := binary.BigEndian.Uint32(data[8:12])
	entries := make([]IndexEntry, 0, entryCount)

	// Direct byte-slice parsing: eliminates binary.Read reflection overhead.
	// Layout per entry (BigEndian for platform-independent portability):
	//  - Ctime/Mtime: 8 bytes each (v2: int64 seconds; v3: {uint32 sec, uint32 nsec})
	//  - Dev/Ino/Mode/Uid/Gid/Size: 4 bytes each
	//  - Sha1: 20 bytes
	//  - Stage flags: 2 bytes
	//  - Path length: 2 bytes
	//  Total fixed: 64 bytes, followed by variable-length path + alignment padding
	pos := 12
	for i := uint32(0); i < entryCount; i++ {
		if pos+64 > len(data) {
			return nil, fmt.Errorf("index truncated at entry %d", i)
		}

		chunk := data[pos:]

		var entry IndexEntry
		if version == 2 {
			// Version 2: timestamps as int64 seconds (nanoseconds lost)
			entry.Ctime = time.Unix(int64(binary.BigEndian.Uint64(chunk[0:8])), 0)
			entry.Mtime = time.Unix(int64(binary.BigEndian.Uint64(chunk[8:16])), 0)
		} else {
			// Version 3: timestamps as {uint32 seconds, uint32 nanoseconds}
			// for nanosecond-accurate stat-cache comparison
			entry.Ctime = time.Unix(
				int64(binary.BigEndian.Uint32(chunk[0:4])),
				int64(binary.BigEndian.Uint32(chunk[4:8])),
			)
			entry.Mtime = time.Unix(
				int64(binary.BigEndian.Uint32(chunk[8:12])),
				int64(binary.BigEndian.Uint32(chunk[12:16])),
			)
		}
		entry.Dev = binary.BigEndian.Uint32(chunk[16:20])
		entry.Ino = binary.BigEndian.Uint32(chunk[20:24])
		entry.Mode = binary.BigEndian.Uint32(chunk[24:28])
		entry.Uid = binary.BigEndian.Uint32(chunk[28:32])
		entry.Gid = binary.BigEndian.Uint32(chunk[32:36])
		entry.Size = binary.BigEndian.Uint32(chunk[36:40])
		copy(entry.Sha1[:], chunk[40:60])
		entry.Stage = binary.BigEndian.Uint16(chunk[60:62])

		pathLen := int(binary.BigEndian.Uint16(chunk[62:64]))

		if pos+64+pathLen > len(data) {
			return nil, fmt.Errorf("index truncated at entry %d path", i)
		}
		entry.Path = string(chunk[64 : 64+pathLen])

		// 8-byte alignment logic:
		// Index formats require entries to be aligned to 8-byte boundary offsets relative
		// to the file start. We compute the exact written bytes for the current entry
		// (62 bytes fixed metadata + 2 bytes path size field + path data length) and
		// advance the position past the alignment padding to the next entry boundary.
		entrySize := 64 + pathLen
		var paddingLen int
		if version == 2 {
			// Version 2: 0-7 bytes padding (original formula)
			paddingLen = (8 - (entrySize % 8)) % 8
		} else {
			// Version 3: 1-8 bytes padding (Git spec compliant, guarantees NUL termination)
			paddingLen = 8 - (entrySize % 8)
		}
		pos += entrySize + paddingLen

		entries = append(entries, entry)
	}
	return entries, nil
}

// WriteIndex serializes the staged index entries back to `.purr/index`.
// It maintains deterministic sorting and layout standards, ensuring that equivalent working states
// generate byte-for-byte identical index files.
//
// Format: Version 3 — timestamps stored as {uint32 seconds, uint32 nanoseconds} pairs,
// padding uses 1-8 NUL bytes (Git spec compliant, guarantees NUL termination).
// Performance: Uses direct PutUint* encoding into a reusable scratch buffer instead of binary.Write,
// eliminating reflection overhead. The buffer is pre-allocated based on expected entry count.
// Crash Safety: Writes to a temporary `.lock` file first, then atomically renames. This prevents
// index corruption if the process crashes mid-write.
func WriteIndex(indexPath string, entries []IndexEntry) error {
	var buf bytes.Buffer
	// Pre-allocate: 12 byte header + ~80 bytes per entry (64 fixed + avg path length)
	buf.Grow(12 + len(entries)*80)

	// Write 12-byte header — version 3 for nanosecond timestamps + spec-compliant padding
	var hdr [12]byte
	copy(hdr[0:4], "DIRC")
	binary.BigEndian.PutUint32(hdr[4:8], 3)
	binary.BigEndian.PutUint32(hdr[8:12], uint32(len(entries)))
	buf.Write(hdr[:])

	// Reusable scratch buffer for the 64-byte fixed-size block per entry
	var fixed [64]byte
	// Reusable zero-padding buffer (max 8 bytes needed for 8-byte alignment with NUL termination)
	var zeroPad [8]byte

	// Write each entry sequentially using direct byte encoding
	for _, entry := range entries {
		// Version 3 timestamps: {uint32 seconds, uint32 nanoseconds} per timestamp
		binary.BigEndian.PutUint32(fixed[0:4], uint32(entry.Ctime.Unix()))
		binary.BigEndian.PutUint32(fixed[4:8], uint32(entry.Ctime.Nanosecond()))
		binary.BigEndian.PutUint32(fixed[8:12], uint32(entry.Mtime.Unix()))
		binary.BigEndian.PutUint32(fixed[12:16], uint32(entry.Mtime.Nanosecond()))
		binary.BigEndian.PutUint32(fixed[16:20], entry.Dev)
		binary.BigEndian.PutUint32(fixed[20:24], entry.Ino)
		binary.BigEndian.PutUint32(fixed[24:28], entry.Mode)
		binary.BigEndian.PutUint32(fixed[28:32], entry.Uid)
		binary.BigEndian.PutUint32(fixed[32:36], entry.Gid)
		binary.BigEndian.PutUint32(fixed[36:40], entry.Size)
		copy(fixed[40:60], entry.Sha1[:])
		binary.BigEndian.PutUint16(fixed[60:62], entry.Stage)

		pathBytes := []byte(entry.Path)
		binary.BigEndian.PutUint16(fixed[62:64], uint16(len(pathBytes)))
		buf.Write(fixed[:])
		buf.Write(pathBytes)

		// 1-8 NUL bytes padding: guarantees NUL termination and 8-byte alignment (Git spec compliant)
		entrySize := 64 + len(pathBytes)
		paddingLen := 8 - (entrySize % 8)
		buf.Write(zeroPad[:paddingLen])
	}

	// Atomic write: temp file + rename prevents index corruption on crash
	tmpPath := indexPath + ".lock"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write index file: %w", err)
	}
	if err := os.Rename(tmpPath, indexPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize index file: %w", err)
	}

	return nil
}


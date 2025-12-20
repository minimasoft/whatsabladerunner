// decode.go - Decode base64 protobuf message samples
// Usage: go run decode.go
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func main() {
	files, err := filepath.Glob("*.b64")
	if err != nil {
		fmt.Printf("Error finding .b64 files: %v\n", err)
		os.Exit(1)
	}

	for _, file := range files {
		fmt.Printf("\n=== Processing %s ===\n", file)
		processFile(file)
	}
}

func processFile(filename string) {
	// Read base64 content
	b64Content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", filename, err)
		return
	}

	b64Str := strings.TrimSpace(string(b64Content))
	if b64Str == "" {
		fmt.Printf("Skipping %s: empty file\n", filename)
		return
	}

	// Decode base64 to binary
	binary, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		fmt.Printf("Error decoding base64: %v\n", err)
		return
	}

	baseName := strings.TrimSuffix(filename, ".b64")

	// Save raw binary
	binFile := baseName + ".bin"
	if err := os.WriteFile(binFile, binary, 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", binFile, err)
	} else {
		fmt.Printf("Saved binary to %s (%d bytes)\n", binFile, len(binary))
	}

	// Try stripping PKCS7 padding
	unpadded := stripPKCS7Padding(binary)
	fmt.Printf("Original: %d bytes, After unpad: %d bytes\n", len(binary), len(unpadded))

	// Raw protobuf decode using protoc --decode_raw
	rawOutput := rawProtobufDecode(unpadded)
	rawFile := baseName + ".raw"
	if err := os.WriteFile(rawFile, []byte(rawOutput), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", rawFile, err)
	} else {
		fmt.Printf("Saved raw decode to %s\n", rawFile)
	}

	// Try to unmarshal as different types
	unmarshalOutput := tryAllUnmarshals(unpadded)
	unmarshalFile := baseName + ".unmarshal"
	if err := os.WriteFile(unmarshalFile, []byte(unmarshalOutput), 0644); err != nil {
		fmt.Printf("Error writing %s: %v\n", unmarshalFile, err)
	} else {
		fmt.Printf("Saved unmarshal to %s\n", unmarshalFile)
	}
}

func stripPKCS7Padding(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	if padding > 0 && padding <= 16 && padding <= len(data) {
		// Verify all padding bytes are the same
		for i := len(data) - padding; i < len(data); i++ {
			if data[i] != byte(padding) {
				return data // Not valid PKCS7 padding
			}
		}
		return data[:len(data)-padding]
	}
	return data
}

func rawProtobufDecode(data []byte) string {
	cmd := exec.Command("protoc", "--decode_raw")
	cmd.Stdin = bytes.NewReader(data)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error running protoc --decode_raw: %v\n\nOutput:\n%s\n\nHex dump:\n%s", err, string(output), hex.Dump(data[:min(256, len(data))]))
	}
	return string(output)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func tryAllUnmarshals(data []byte) string {
	var sb strings.Builder

	// Show first bytes to understand the structure
	sb.WriteString("=== First 32 bytes (hex) ===\n")
	sb.WriteString(hex.Dump(data[:min(32, len(data))]))
	sb.WriteString("\n")

	// Try as waE2E.Message directly
	sb.WriteString("=== Try 1: waE2E.Message (direct) ===\n")
	msg := &waProto.Message{}
	if err := proto.Unmarshal(data, msg); err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n\n", err))
	} else {
		sb.WriteString("SUCCESS!\n")
		sb.WriteString(prototext.Format(msg))
		sb.WriteString("\n")
	}

	// Check if first byte indicates a known field
	if len(data) > 2 {
		firstByte := data[0]
		fieldNum := firstByte >> 3
		wireType := firstByte & 0x07
		sb.WriteString(fmt.Sprintf("=== Wire analysis: first field=%d, wire_type=%d ===\n", fieldNum, wireType))

		// Field 19 (0x9a >> 3 = 19) = MessageContextInfo in waE2E.Message
		// Field 31 (0xfa >> 3 = 31) = likely some container
		// Field 27 (0xda >> 3 = 27) = buttonsResponseMessage

		// If field 19 (MessageContextInfo), this IS a waE2E.Message
		if fieldNum == 19 || fieldNum == 27 {
			sb.WriteString("This appears to be a waE2E.Message (has MessageContextInfo or ButtonsResponse)\n\n")
		}

		// Try skipping length-prefixed envelope if field 31
		if fieldNum == 31 && wireType == 2 {
			sb.WriteString("=== Try 2: Skip outer envelope (field 31) ===\n")
			// Read varint length
			offset := 1
			length, n := decodeVarint(data[offset:])
			if n > 0 {
				offset += n
				innerData := data[offset : offset+int(length)]
				sb.WriteString(fmt.Sprintf("Inner message: %d bytes\n", len(innerData)))

				msg2 := &waProto.Message{}
				if err := proto.Unmarshal(innerData, msg2); err != nil {
					sb.WriteString(fmt.Sprintf("Error: %v\n\n", err))
					// Try raw decode of inner
					sb.WriteString("Raw decode of inner:\n")
					sb.WriteString(rawProtobufDecode(innerData))
				} else {
					sb.WriteString("SUCCESS!\n")
					sb.WriteString(prototext.Format(msg2))
				}
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

func decodeVarint(data []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, b := range data {
		if b < 0x80 {
			return x | uint64(b)<<s, i + 1
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0
}

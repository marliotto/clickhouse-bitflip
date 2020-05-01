package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/lib/cityhash102"
	"io"
	"os"
)

type ClickHouseChecksum struct {
	First, Second uint64
}

func (c ClickHouseChecksum) String() string {
	return fmt.Sprintf("%x%x", c.First, c.Second)
}

type ClickHouseHeader struct {
	Method                           uint8
	CompressedSize, UncompressedSize uint32
}

var uncorrectedErrors int32 = 0
var correctedErrors int32 = 0

func main() {
	flag.Parse()
	srcFile := flag.Arg(0)

	f, err := os.OpenFile(srcFile, os.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for {
		expected := readChecksum(f)
		//fmt.Println("Reference:", expected)

		startPosOfBlock, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			panic(err)
		}

		data, _ := readData(f)
		//fmt.Println("Size of compressed block:", header.CompressedSize)

		bitFlipPos, fixed := fixData(data, expected)
		if fixed {
			backup(srcFile)
			fixFile(f, data, int64(bitFlipPos), startPosOfBlock)
		}
	}
}

func fixFile(f *os.File, data []byte, bitPos int64, startPosOfBlock int64) {
	pos := bitPos / 8
	filePos := startPosOfBlock + pos

	fmt.Printf("    Fixing in file at byte %v. Old value: 0x%x. New value: 0x%x\n", filePos, data[pos]^1<<(bitPos%8), data[pos])

	_, err := f.WriteAt(data[pos:pos+1], filePos)
	if err != nil {
		panic(err)
	}
	correctedErrors++
}

func fixData(data []byte, expected ClickHouseChecksum) (int, bool) {
	isEqual, actual := compareChecksum(data, expected)
	if isEqual {
		//fmt.Printf("    Checksums are equal. Reference: %v\n", expected)
		return 0, false
	}
	fmt.Printf("Checksum doesn't match: corrupted data. Reference: %v. Actual: %v. Size of compressed block: %v\n", expected, actual, len(data))

	for bitPos := 0; bitPos < len(data)*8; bitPos++ {
		flipBit(data, bitPos)
		isEqual, actual = compareChecksum(data, expected)
		if isEqual {
			fmt.Println("    The mismatch is caused by single bit flip in data block at byte", bitPos/8, ", bit", bitPos%8)
			return bitPos, true
		}
		flipBit(data, bitPos)
	}

	uncorrectedErrors++
	fmt.Println("    Error: the mismatch is not caused by single bit flip. It can't be fixed!")
	return 0, false
}

func flipBit(data []byte, pos int) {
	data[pos/8] ^= 1 << (pos % 8)
}

func readChecksum(f *os.File) ClickHouseChecksum {
	var checksum ClickHouseChecksum
	err := binary.Read(f, binary.LittleEndian, &checksum)
	if err == io.EOF {
		if uncorrectedErrors == 0 && correctedErrors == 0 {
			fmt.Println("No errors")
		} else {
			fmt.Println("Completed. Corrected errors:", correctedErrors, ". Uncorrected errors", uncorrectedErrors)
		}
		os.Exit(0)
	}
	if err != nil {
		fmt.Println("binary.Read failed:", err)
		os.Exit(1)
	}

	return checksum
}

func readData(f *os.File) (data []byte, header ClickHouseHeader) {
	err := binary.Read(f, binary.LittleEndian, &header)
	if err != nil {
		fmt.Println("binary.Read failed:", err)
		os.Exit(1)
	}
	var headerSize = binary.Size(header)
	_, err = f.Seek(int64(headerSize*(-1)), io.SeekCurrent) // go back, checksum contains header
	if err != nil {
		fmt.Println("f.Seek failed:", err)
		os.Exit(1)
	}
	data = make([]byte, header.CompressedSize)
	n, err := f.Read(data)
	if err != nil {
		fmt.Println("f.Read failed:", err)
		os.Exit(1)
	}
	if n != int(header.CompressedSize) {
		fmt.Printf("error: file is clipped, expected compressed size: %v, actual: %v\n", header.CompressedSize, n)
		os.Exit(1)
	}

	return
}

func backup(src string) {
	dst := src + ".bak"
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		fmt.Println("    backup file to", dst)
		_, err = copyFile(src, dst)
		if err != nil {
			panic(err)
		}
	}
}

func copyFile(src string, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func compareChecksum(data []byte, expected ClickHouseChecksum) (result bool, actual ClickHouseChecksum) {
	actualUint128 := cityhash102.CityHash128(data, uint32(len(data)))
	actual = ClickHouseChecksum{actualUint128[0], actualUint128[1]}
	result = expected == actual

	return
}

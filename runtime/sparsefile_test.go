package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const testImageSize = 1024 * 1024 * 1024 // 1GB

func createExt4Image(path string, size int) error {
	err := createSparseFile(path, size)
	if err != nil {
		return err
	}

	blockSize := 1024
	mkfs := exec.Command(
		"mkfs.ext4",
		"-F",
		"-b", strconv.Itoa(blockSize),
		path,
		strconv.Itoa(int(size/blockSize)),
	)
	_, err = mkfs.CombinedOutput()
	return err
}

func benchmarkCopyGo(b *testing.B, create func(string, int) error) {
	dir, err := ioutil.TempDir("", b.Name())
	require.NoError(b, err)
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "src")
	err = create(src, testImageSize)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		dst := filepath.Join(dir, fmt.Sprintf("dst%d", i))
		err = copyFile(src, dst, 0600)
		require.NoError(b, err)
	}
}

func BenchmarkCopyGoEmptyFile(b *testing.B) {
	benchmarkCopyGo(b, createSparseFile)
}

func BenchmarkCopyGoExt4(b *testing.B) {
	benchmarkCopyGo(b, createExt4Image)
}

func copyFileNaive(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	io.Copy(dst, src)
	return nil
}

func benchmarkCopyGoNaive(b *testing.B, create func(string, int) error) {
	dir, err := ioutil.TempDir("", b.Name())
	require.NoError(b, err)
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "src")
	err = create(src, testImageSize)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		dst := filepath.Join(dir, fmt.Sprintf("dst%d", i))
		err = copyFileNaive(src, dst)
		require.NoError(b, err)
	}
}

func BenchmarkCopyGoNaiveEmptyFile(b *testing.B) {
	benchmarkCopyGoNaive(b, createSparseFile)
}

func BenchmarkCopyGoNaiveExt4(b *testing.B) {
	benchmarkCopyGoNaive(b, createExt4Image)
}

func benchmarkCopyExec(b *testing.B, create func(string, int) error) {
	dir, err := ioutil.TempDir("", b.Name())
	require.NoError(b, err)
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "src")
	err = create(src, testImageSize)
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		dst := filepath.Join(dir, fmt.Sprintf("dst%d", i))
		cp := exec.Command("cp", "--sparse=always", src, dst)
		_, err := cp.CombinedOutput()
		require.NoError(b, err)
	}
}

func BenchmarkCopyExecEmptyFile(b *testing.B) {
	benchmarkCopyExec(b, createSparseFile)
}

func BenchmarkCopyExecExt4(b *testing.B) {
	benchmarkCopyExec(b, createExt4Image)
}

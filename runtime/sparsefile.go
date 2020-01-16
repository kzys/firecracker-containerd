// Copyright 2018-2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"io"
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// SEEK_DATA and SEEK_HOLE were originally implemented in Solaris and later adopted by Linux (since 3.1)
// https://lwn.net/Articles/440255/
// http://man7.org/linux/man-pages/man2/lseek.2.html
const (
	seekData = 3
	seekHole = 4
	eNXIO    = 6
)

func isENXIO(err error) bool {
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno == eNXIO
		}
	}
	return false
}

func copySparseFile(dst, src *os.File) error {
	var offset int64

	srcEnd, err := src.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	for offset < srcEnd {
		// Find an region that contain data
		begin, err := src.Seek(offset, seekData)
		if err != nil {
			if isENXIO(err) {
				break
			}
			return errors.Wrapf(err, "failed to find data after %d", offset)
		}

		// Find a hole, which is the end of data
		end, err := src.Seek(begin, seekHole)
		if err != nil {
			// We don't have to check ENXIO here since there is an implicit hole at the end of any file
			return errors.Wrapf(err, "failed to find a hole after %d", offset)
		}

		// Back to the data region again to copy
		_, err = src.Seek(begin, io.SeekStart)
		if err != nil {
			return err
		}
		_, err = dst.Seek(begin, io.SeekStart)
		if err != nil {
			return err
		}

		// Copy data
		_, err = io.CopyN(dst, src, end-begin)
		if err != nil {
			return err
		}

		offset = end
	}

	return dst.Truncate(srcEnd)
}

func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.Wrapf(err, "failed to open %v", src)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, mode)
	if err != nil {
		return errors.Wrapf(err, "failed to open %v", dstFile)
	}
	defer dstFile.Close()

	err = copySparseFile(dstFile, srcFile)
	if err != nil {
		return errors.Wrap(err, "failed to copy to destination")
	}
	return nil
}

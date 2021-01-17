// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package filesstore

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/utils"
)

type FileBackendTestSuite struct {
	suite.Suite

	settings model.FileSettings
	backend  FileBackend
}

func TestLocalFileBackendTestSuite(t *testing.T) {
	// Setup a global logger to catch tests logging outside of app context
	// The global logger will be stomped by apps initializing but that's fine for testing. Ideally this won't happen.
	mlog.InitGlobalLogger(mlog.NewLogger(&mlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleJson:   true,
		ConsoleLevel:  "error",
		EnableFile:    false,
	}))

	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	suite.Run(t, &FileBackendTestSuite{
		settings: model.FileSettings{
			DriverName: model.NewString(model.IMAGE_DRIVER_LOCAL),
			Directory:  &dir,
		},
	})
}

func TestS3FileBackendTestSuite(t *testing.T) {
	runBackendTest(t, false)
}

func TestS3FileBackendTestSuiteWithEncryption(t *testing.T) {
	runBackendTest(t, true)
}

func runBackendTest(t *testing.T, encrypt bool) {
	s3Host := os.Getenv("CI_MINIO_HOST")
	if s3Host == "" {
		s3Host = "localhost"
	}

	s3Port := os.Getenv("CI_MINIO_PORT")
	if s3Port == "" {
		s3Port = "9000"
	}

	s3Endpoint := fmt.Sprintf("%s:%s", s3Host, s3Port)

	suite.Run(t, &FileBackendTestSuite{
		settings: model.FileSettings{
			DriverName:              model.NewString(model.IMAGE_DRIVER_S3),
			AmazonS3AccessKeyId:     model.NewString(model.MINIO_ACCESS_KEY),
			AmazonS3SecretAccessKey: model.NewString(model.MINIO_SECRET_KEY),
			AmazonS3Bucket:          model.NewString(model.MINIO_BUCKET),
			AmazonS3Region:          model.NewString(""),
			AmazonS3Endpoint:        model.NewString(s3Endpoint),
			AmazonS3PathPrefix:      model.NewString(""),
			AmazonS3SSL:             model.NewBool(false),
			AmazonS3SSE:             model.NewBool(encrypt),
		},
	})
}

func (s *FileBackendTestSuite) SetupTest() {
	utils.TranslationsPreInit()

	backend, err := NewFileBackend(&s.settings, true)
	require.Nil(s.T(), err)
	s.backend = backend

	// This is needed to create the bucket if it doesn't exist.
	s.Nil(s.backend.TestConnection())
}

func (s *FileBackendTestSuite) TestConnection() {
	s.Nil(s.backend.TestConnection())
}

func (s *FileBackendTestSuite) TestReadWriteFile() {
	b := []byte("test")
	path := "tests/" + model.NewId()

	written, err := s.backend.WriteFile(bytes.NewReader(b), path)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")
	defer s.backend.RemoveFile(path)

	read, err := s.backend.ReadFile(path)
	s.Nil(err)

	readString := string(read)
	s.EqualValues(readString, "test")
}

func (s *FileBackendTestSuite) TestReadWriteFileImage() {
	b := []byte("testimage")
	path := "tests/" + model.NewId() + ".png"

	written, err := s.backend.WriteFile(bytes.NewReader(b), path)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")
	defer s.backend.RemoveFile(path)

	read, err := s.backend.ReadFile(path)
	s.Nil(err)

	readString := string(read)
	s.EqualValues(readString, "testimage")
}

func (s *FileBackendTestSuite) TestFileExists() {
	b := []byte("testimage")
	path := "tests/" + model.NewId() + ".png"

	_, err := s.backend.WriteFile(bytes.NewReader(b), path)
	s.Nil(err)
	defer s.backend.RemoveFile(path)

	res, err := s.backend.FileExists(path)
	s.Nil(err)
	s.True(res)

	res, err = s.backend.FileExists("tests/idontexist.png")
	s.Nil(err)
	s.False(res)
}

func (s *FileBackendTestSuite) TestCopyFile() {
	b := []byte("test")
	path1 := "tests/" + model.NewId()
	path2 := "tests/" + model.NewId()

	written, err := s.backend.WriteFile(bytes.NewReader(b), path1)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")
	defer s.backend.RemoveFile(path1)

	err = s.backend.CopyFile(path1, path2)
	s.Nil(err)
	defer s.backend.RemoveFile(path2)

	data1, err := s.backend.ReadFile(path1)
	s.Nil(err)

	data2, err := s.backend.ReadFile(path2)
	s.Nil(err)

	s.Equal(b, data1)
	s.Equal(b, data2)
}

func (s *FileBackendTestSuite) TestCopyFileToDirectoryThatDoesntExist() {
	b := []byte("test")
	path1 := "tests/" + model.NewId()
	path2 := "tests/newdirectory/" + model.NewId()

	written, err := s.backend.WriteFile(bytes.NewReader(b), path1)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")
	defer s.backend.RemoveFile(path1)

	err = s.backend.CopyFile(path1, path2)
	s.Nil(err)
	defer s.backend.RemoveFile(path2)

	_, err = s.backend.ReadFile(path1)
	s.Nil(err)

	_, err = s.backend.ReadFile(path2)
	s.Nil(err)
}

func (s *FileBackendTestSuite) TestMoveFile() {
	b := []byte("test")
	path1 := "tests/" + model.NewId()
	path2 := "tests/" + model.NewId()

	written, err := s.backend.WriteFile(bytes.NewReader(b), path1)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")
	defer s.backend.RemoveFile(path1)

	s.Nil(s.backend.MoveFile(path1, path2))
	defer s.backend.RemoveFile(path2)

	_, err = s.backend.ReadFile(path1)
	s.Error(err)

	data, err := s.backend.ReadFile(path2)
	s.Nil(err)

	s.Equal(b, data)
}

func (s *FileBackendTestSuite) TestRemoveFile() {
	b := []byte("test")
	path := "tests/" + model.NewId()

	written, err := s.backend.WriteFile(bytes.NewReader(b), path)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")
	s.Nil(s.backend.RemoveFile(path))

	_, err = s.backend.ReadFile(path)
	s.Error(err)

	written, err = s.backend.WriteFile(bytes.NewReader(b), "tests2/foo")
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	written, err = s.backend.WriteFile(bytes.NewReader(b), "tests2/bar")
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	written, err = s.backend.WriteFile(bytes.NewReader(b), "tests2/asdf")
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	s.Nil(s.backend.RemoveDirectory("tests2"))
}

func (s *FileBackendTestSuite) TestListDirectory() {
	b := []byte("test")
	path1 := "19700101/" + model.NewId()
	path2 := "19800101/" + model.NewId()

	paths, err := s.backend.ListDirectory("19700101")
	s.Nil(err)
	s.Len(*paths, 0)

	written, err := s.backend.WriteFile(bytes.NewReader(b), path1)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	written, err = s.backend.WriteFile(bytes.NewReader(b), path2)
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	paths, err = s.backend.ListDirectory("19700101")
	s.Nil(err)
	s.Len(*paths, 1)
	s.Equal(path1, (*paths)[0])

	paths, err = s.backend.ListDirectory("19700101/")
	s.Nil(err)
	s.Len(*paths, 1)
	s.Equal(path1, (*paths)[0])

	paths, err = s.backend.ListDirectory("")
	s.Nil(err)

	found1 := false
	found2 := false
	for _, path := range *paths {
		if path == "19700101" {
			found1 = true
		} else if path == "19800101" {
			found2 = true
		}
	}
	s.True(found1)
	s.True(found2)

	s.backend.RemoveFile(path1)
	s.backend.RemoveFile(path2)
}

func (s *FileBackendTestSuite) TestRemoveDirectory() {
	b := []byte("test")

	written, err := s.backend.WriteFile(bytes.NewReader(b), "tests2/foo")
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	written, err = s.backend.WriteFile(bytes.NewReader(b), "tests2/bar")
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	written, err = s.backend.WriteFile(bytes.NewReader(b), "tests2/aaa")
	s.Nil(err)
	s.EqualValues(len(b), written, "expected given number of bytes to have been written")

	s.Nil(s.backend.RemoveDirectory("tests2"))

	_, err = s.backend.ReadFile("tests2/foo")
	s.Error(err)
	_, err = s.backend.ReadFile("tests2/bar")
	s.Error(err)
	_, err = s.backend.ReadFile("tests2/asdf")
	s.Error(err)
}

func (s *FileBackendTestSuite) TestAppendFile() {
	s.Run("should fail if target file is missing", func() {
		path := "tests/" + model.NewId()
		b := make([]byte, 1024)
		written, err := s.backend.AppendFile(bytes.NewReader(b), path)
		s.Error(err)
		s.Zero(written)
	})

	s.Run("should correctly append the data", func() {
		// First part needs to be at least 5MB for the S3 implementation to work.
		size := 5 * 1024 * 1024
		b := make([]byte, size)
		for i := range b {
			b[i] = 'A'
		}
		path := "tests/" + model.NewId()

		written, err := s.backend.WriteFile(bytes.NewReader(b), path)
		s.Nil(err)
		s.EqualValues(len(b), written)
		defer s.backend.RemoveFile(path)

		b2 := make([]byte, 1024)
		for i := range b2 {
			b2[i] = 'B'
		}

		written, err = s.backend.AppendFile(bytes.NewReader(b2), path)
		s.Nil(err)
		s.EqualValues(int64(len(b2)), written)

		read, err := s.backend.ReadFile(path)
		s.Nil(err)
		s.EqualValues(len(b)+len(b2), len(read))
		s.EqualValues(append(b, b2...), read)

		b3 := make([]byte, 1024)
		for i := range b3 {
			b3[i] = 'C'
		}

		written, err = s.backend.AppendFile(bytes.NewReader(b3), path)
		s.Nil(err)
		s.EqualValues(int64(len(b3)), written)

		read, err = s.backend.ReadFile(path)
		s.Nil(err)
		s.EqualValues(len(b)+len(b2)+len(b3), len(read))
		s.EqualValues(append(append(b, b2...), b3...), read)
	})
}

func (s *FileBackendTestSuite) TestFileSize() {
	s.Run("nonexistent file", func() {
		size, err := s.backend.FileSize("tests/nonexistentfile")
		s.NotNil(err)
		s.Zero(size)
	})

	s.Run("valid file", func() {
		data := make([]byte, rand.Intn(1024*1024)+1)
		path := "tests/" + model.NewId()

		written, err := s.backend.WriteFile(bytes.NewReader(data), path)
		s.Nil(err)
		s.EqualValues(len(data), written)
		defer s.backend.RemoveFile(path)

		size, err := s.backend.FileSize(path)
		s.Nil(err)
		s.Equal(int64(len(data)), size)
	})
}

func BenchmarkS3WriteFile(b *testing.B) {
	utils.TranslationsPreInit()

	settings := &model.FileSettings{
		DriverName:              model.NewString(model.IMAGE_DRIVER_S3),
		AmazonS3AccessKeyId:     model.NewString(model.MINIO_ACCESS_KEY),
		AmazonS3SecretAccessKey: model.NewString(model.MINIO_SECRET_KEY),
		AmazonS3Bucket:          model.NewString(model.MINIO_BUCKET),
		AmazonS3Region:          model.NewString(""),
		AmazonS3Endpoint:        model.NewString("localhost:9000"),
		AmazonS3PathPrefix:      model.NewString(""),
		AmazonS3SSL:             model.NewBool(false),
		AmazonS3SSE:             model.NewBool(false),
	}

	backend, err := NewFileBackend(settings, true)
	require.Nil(b, err)

	// This is needed to create the bucket if it doesn't exist.
	require.Nil(b, backend.TestConnection())

	path := "tests/" + model.NewId()
	size := 1 * 1024 * 1024
	data := make([]byte, size)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		written, err := backend.WriteFile(bytes.NewReader(data), path)
		defer backend.RemoveFile(path)
		require.Nil(b, err)
		require.Equal(b, len(data), int(written))
	}

	b.StopTimer()
}

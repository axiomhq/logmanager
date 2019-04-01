package logmanager

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskWriter(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	marker1 := "yeahhhhh boiiiii"
	marker2 := "than you, far too kind"
	marker3 := "i am potato"

	tmpDir, err := ioutil.TempDir(os.TempDir(), "testdiskwriter")
	defer os.RemoveAll(tmpDir) // nolint
	require.NoError(err)

	logPath := path.Join(tmpDir, "logfile.log")
	writer := NewDiskWriter(logPath, DiskWriterConfig{
		RotateDuration:  time.Millisecond,
		MaximumLogFiles: 3,
	})

	writer.Log(Info, ColorTheme{}, "logmanager", "diskwriter_test.go", 32, time.Now(), "whatmahnameeee")
	<-time.After(time.Millisecond * 10)

	_, err = os.Stat(logPath)
	require.NoError(err, "log file should exist after logging")
	writer.Close()

	writer = NewDiskWriter(logPath, DiskWriterConfig{
		RotateDuration:  time.Millisecond,
		MaximumLogFiles: 3,
	})

	<-time.After(time.Millisecond * 10)
	writer.Log(Info, ColorTheme{}, "logmanager", "diskwriter_test.go", 45, time.Now(), marker1)
	<-time.After(time.Millisecond * 10)

	_, err = os.Stat(logPath)
	assert.NoError(err, "log file should still exist after a second writer is created")

	_, err = os.Stat(logPath + ".1")
	require.NoError(err, "logfile should now have rotated once")

	writer.Log(Info, ColorTheme{}, "logmanager", "diskwriter_test.go", 54, time.Now(), marker2)
	<-time.After(time.Millisecond * 10)

	_, err = os.Stat(logPath)
	assert.NoError(err, "log file should still exist after a rotation")

	_, err = os.Stat(logPath + ".1")
	assert.NoError(err, "second log file should still exist")

	_, err = os.Stat(logPath + ".2")
	require.NoError(err, "third log file should now exist")

	writer.Log(Info, ColorTheme{}, "logmanager", "diskwriter_test.go", 66, time.Now(), marker3)
	<-time.After(time.Millisecond * 10)

	_, err = os.Stat(logPath + ".3")
	require.Error(err, "after the third rotation we hit max log files (3), .3 should not exist")
	writer.Close()

	all, err := ioutil.ReadFile(logPath)
	require.NoError(err)
	assert.Contains(string(all), marker3, "logfile should contain last written message")

	all, err = ioutil.ReadFile(logPath + ".1")
	require.NoError(err)
	assert.Contains(string(all), marker2, "logfile.1 should contain the correct message")

	all, err = ioutil.ReadFile(logPath + ".2")
	require.NoError(err)
	assert.Contains(string(all), marker1, "logfile.2 should contain the correct message")
}

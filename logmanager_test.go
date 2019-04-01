package logmanager

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// doesn't actually test the logger, just lets us see what it looks like
func TestLookAtTheLogFormat(t *testing.T) {
	logmodules := []Logger{
		GetLogger("MODULEred"),
		GetLogger("MODUleYellow"),
		GetLogger("ModuLECYAN"),
		GetLogger("ModuLEGREEN"),
		GetLogger("ModuleBLue"),
		GetLogger("ModuleMagenTA"),
		GetLogger("ModuleWhite"),
	}

	// also read-add some of the log modules to test race conditions
	for i := 0; i <= 10; i++ {
		logmodules = append(logmodules, logmodules[0])
	}

	var wg sync.WaitGroup
	for _, log := range logmodules {
		wg.Add(1)
		go func(logger Logger) {
			logger.Trace("test trace")
			logger.Debug("test debug")
			logger.Info("test info")
			logger.Warn("test warning")
			logger.Error("test error")
			logger.Critical("test critical")
			wg.Done()
		}(log)
	}
	wg.Wait()

	err := errors.New("test error")
	if logmodules[0].IsError(err) == false {
		t.Fail()
	}

	retErr := logmodules[0].Error("testError %s", err)
	assert.EqualError(t, retErr, err.Error())

	retErr = logmodules[0].Error("testError %s %s", "foo", err)
	assert.EqualError(t, retErr, err.Error())

	retErr = logmodules[0].Error("testError %s %s %s", "different thing", err, "foobar")
	assert.EqualError(t, retErr, "testError different thing test error foobar")
}

func TestIsError(t *testing.T) {
	logger := GetLogger("fuck.you.gord")

	assert.False(t, logger.IsError(func() error {
		return nil
	}()))

	assert.True(t, logger.IsError(func() error {
		return errors.New("fuck")
	}()))
}

func TestRecover(t *testing.T) {
	logger := GetLogger("no.fuck.you")
	assert := assert.New(t)
	require := require.New(t)

	check := func(panicfunc func()) (err error) {
		defer func() {
			if r := logger.Recover(recover()); r != nil {
				err = r
			}
		}()

		panicfunc()
		return //nolint(vet)
	}

	checkPanicErr := func() error { return check(func() { panic(errors.New("oh no")) }) }
	checkPanicString := func() error { return check(func() { panic("oh no") }) }
	checkNoPanic := func() error { return check(func() {}) }

	require.NotPanics(func() { checkPanicErr() })
	assert.EqualError(checkPanicErr(), "oh no")
	assert.EqualError(checkPanicString(), "oh no")

	require.NotPanics(func() { checkNoPanic() })
	assert.NoError(checkNoPanic())
}

func TestLogAndExtend(t *testing.T) {
	logger := GetLogger("panic-test")

	logger.Info("Logging happily")
	SetCustomWriters(NewConsoleWriter())
	logger.Info("Crashing... nah, got fixed")
}

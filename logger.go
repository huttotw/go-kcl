package kcl

// Logger is an interface that helps the user to log what is happening
// inside of this pacakge. This interface is pretty common and is used by
// https://github.com/go-kit/kit
type Logger interface {
	Log(keyvals ...interface{}) error
}

// noOpLogger allows us to not log things by default, simply set the
// Logger on the Stream struct if you wish to start logging messages.
type noOpLogger struct{}

func (l noOpLogger) Log(keyvals ...interface{}) error {
	return nil
}

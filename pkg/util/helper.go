package util

import (
	"bufio"
	"context"
	"crypto/x509"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

var (

	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
)

// RetryPolicyOn404 is similar to the defaulRetryPolicy but
// which an additional check for 404 status.
func RetryPolicyOn404(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, nil
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	// 429 Too Many Requests is recoverable. Sometimes the server puts
	// a Retry-After response header to indicate when the server is
	// available to start processing request from client.
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || resp.StatusCode == 400 || resp.StatusCode == 404 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, nil
	}

	return false, nil
}

// AskBool function asks for the user input
// for a boolean input
func AskBool(msg string, args ...interface{}) (bool, error) {
	_, err := fmt.Fprintf(os.Stdout, fmt.Sprintf("%s (y/n): ", msg), args...)
	if err != nil {
		return false, fmt.Errorf("Unable to show options to user: %s", err.Error())
	}

	r := bufio.NewReader(os.Stdin)
	byt, isPrefix, err := r.ReadLine()

	if isPrefix || err != nil {
		return false, fmt.Errorf("Unable to read i/p: %s", err.Error())
	}

	resp := string(byt)
	if resp == "y" || resp == "Y" {
		return true, nil
	}

	if resp == "n" || resp == "N" {
		return false, nil
	}

	return false, fmt.Errorf("Please provide input as y or n, provided: %s", resp)
}

// Logger interface allows to use other loggers than
// standard log.Logger.
type ZapWrapper struct {
}

/*
	Implmenting the LeveledLogger for retry http
	type LeveledLogger interface {
		Error(msg string, keysAndValues ...interface{})
		Info(msg string, keysAndValues ...interface{})
		Debug(msg string, keysAndValues ...interface{})
		Warn(msg string, keysAndValues ...interface{})
	}
*/
func (z *ZapWrapper) Error(msg string, args ...interface{}) {
	zap.S().Errorf(msg, args)
}

func (z *ZapWrapper) Info(msg string, args ...interface{}) {
	zap.S().Infof(msg, args)
}

func (z *ZapWrapper) Debug(msg string, args ...interface{}) {
	zap.S().Debugf(msg, args)
}

func (z *ZapWrapper) Warn(msg string, args ...interface{}) {
	zap.S().Warnf(msg, args)
}

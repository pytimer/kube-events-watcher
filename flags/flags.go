package flags

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

/*
	String() string
	Set(string) error
	Type() string
*/

type Uri struct {
	Key string
	Val url.URL
}

func (u *Uri) String() string {
	val := u.Val.String()
	if val == "" {
		return fmt.Sprintf("%s", u.Key)
	}
	return fmt.Sprintf("%s:%s", u.Key, val)
}

func (u *Uri) Set(value string) error {
	s := strings.SplitN(value, ":", 2)
	if s[0] == "" {
		return fmt.Errorf("missing uri key in '%s'", value)
	}
	u.Key = s[0]
	if len(s) > 1 && s[1] != "" {
		e := os.ExpandEnv(s[1])
		uri, err := url.Parse(e)
		if err != nil {
			return err
		}
		u.Val = *uri
	}
	return nil
}

func (u *Uri) Type() string {
	return fmt.Sprintf("%T", u)
}

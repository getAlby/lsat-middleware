package caveat

import (
	"fmt"
	"strings"

	"gopkg.in/macaroon.v2"
)

type Caveat struct {
	Condition string
	Value     string
}

func EncodeCaveat(caveat Caveat) string {
	return fmt.Sprintf("%s=%s", caveat.Condition, caveat.Value)
}

func DecodeCaveat(caveatString string) (Caveat, error) {
	splitted := strings.Split(caveatString, "=")
	if len(splitted) != 2 {
		return Caveat{}, fmt.Errorf("LSAT does not have the right format: %s", caveatString)
	}
	return Caveat{Condition: splitted[0], Value: splitted[1]}, nil
}

func AddFirstPartyCaveats(mac *macaroon.Macaroon, caveats []Caveat) error {
	for _, c := range caveats {
		rawCaveat := []byte(EncodeCaveat(c))
		if err := mac.AddFirstPartyCaveat(rawCaveat); err != nil {
			return err
		}
	}
	return nil
}

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

func VerifyCaveats(rawCaveats []string, conditions []Caveat) error {
	caveats := make([]Caveat, 0, len(rawCaveats))
	for _, rawCaveat := range rawCaveats {
		caveat, err := DecodeCaveat(rawCaveat)
		// Continue to avoid failing if contains any third party caveats
		if err != nil {
			continue
		}
		caveats = append(caveats, caveat)
	}
	if !CheckCaveatsArrayUnorderedEqual(caveats, conditions) {
		return fmt.Errorf("Caveats does not match")
	}
	return nil
}

func CheckCaveatsArrayUnorderedEqual(caveats []Caveat, conditions []Caveat) bool {
	conditionsExist := make(map[string]Caveat, len(conditions))
	for _, condition := range conditions {
		conditionsExist[condition.Condition] = condition
	}
	for _, caveat := range caveats {
		if _, ok := conditionsExist[caveat.Condition]; !ok {
			return false
		}
		if conditionsExist[caveat.Condition].Value != caveat.Value {
			return false
		}
	}
	return true
}

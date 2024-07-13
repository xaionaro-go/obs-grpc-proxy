package obsdoc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
)

func ParseProtocol(b []byte) (*Protocol, error) {
	var p Protocol
	err := json.Unmarshal(b, &p)
	if err != nil {
		return nil, fmt.Errorf("unable to parse the protocol description: %w", err)
	}
	err = fixEnumValues(&p)
	if err != nil {
		return nil, fmt.Errorf("unable to fix the enum values: %w", err)
	}
	return &p, nil
}

func fixEnumValues(p *Protocol) error {
	for enumIdx, enum := range p.Enums {
		for rowIdx, row := range enum.EnumIdentifiers {
			switch v := row.EnumValue.(type) {
			case string:
				if fixed, ok := tryFixEnumValueTypeName(v, row, rowIdx); ok {
					p.Enums[enumIdx].EnumIdentifiers[rowIdx].EnumValue = fixed
					continue
				}
				if fixed, ok, err := tryFixEnumValueBitShift(v); ok {
					if err != nil {
						return fmt.Errorf("unable to parse the bit-shift value of row #%d (%s) of enum #%d (%s): %w", rowIdx, row.EnumIdentifier, enumIdx, enum.EnumType, err)
					}
					p.Enums[enumIdx].EnumIdentifiers[rowIdx].EnumValue = fixed
					continue
				}
				if fixed, ok, err := tryFixEnumValueRef(v, p.Enums[enumIdx].EnumIdentifiers); ok {
					if err != nil {
						return fmt.Errorf("unable to parse the ref-value of row #%d (%s) of enum #%d (%s): %w", rowIdx, row.EnumIdentifier, enumIdx, enum.EnumType, err)
					}
					p.Enums[enumIdx].EnumIdentifiers[rowIdx].EnumValue = fixed
					continue
				}
			case float64:
				p.Enums[enumIdx].EnumIdentifiers[rowIdx].EnumValue = int64(v)
			}
		}
	}
	return nil
}

func tryFixEnumValueTypeName(v string, row EnumIdentifier, rowIdx int) (int64, bool) {
	if row.EnumIdentifier == v {
		return int64(rowIdx), true
	}
	return 0, false
}

var enumBitShiftValue = regexp.MustCompile(`\((\d+) << (\d+)\)`)

func tryFixEnumValueBitShift(v string) (int64, bool, error) {
	match := enumBitShiftValue.FindAllStringSubmatch(v, -1)
	if match == nil {
		return 0, false, nil
	}
	significandStr := match[0][1]
	exponentStr := match[0][2]
	significand, err := strconv.ParseInt(significandStr, 10, 64)
	if err != nil {
		return 0, true, fmt.Errorf("unable to parse significand '%s' as an integer: %w", significandStr, err)
	}
	exponent, err := strconv.ParseUint(exponentStr, 10, 64)
	if err != nil {
		return 0, true, fmt.Errorf("unable to parse exponent '%s' as an unsigned integer: %w", exponentStr, err)
	}

	return significand << uint(exponent), true, nil
}

// example: (General | Config | Scenes | Inputs | Transitions | Filters | Outputs | SceneItems | MediaInputs | Vendors | Ui)
var enumRefValueChecker = regexp.MustCompile(`\((\w+( \| )?)+\)`)
var enumRefValue = regexp.MustCompile(`\w+`)

func tryFixEnumValueRef(v string, rows []EnumIdentifier) (int64, bool, error) {
	match := enumRefValueChecker.FindAllStringSubmatch(v, -1)
	if match == nil {
		return 0, false, nil
	}

	match = enumRefValue.FindAllStringSubmatch(v, -1)
	if match == nil {
		return 0, false, fmt.Errorf("cannot find a single value")
	}

	rowsMap := map[string]*EnumIdentifier{}
	for idx := range rows {
		row := &rows[idx]
		rowsMap[row.EnumIdentifier] = row
	}

	var sum int64 = 0
	for _, submatch := range match {
		ref := submatch[0]
		part, ok := rowsMap[ref]
		if !ok {
			return 0, true, fmt.Errorf("unable to find the enum-identifier '%s' (all parts: %v)", ref, match)
		}
		switch v := part.EnumValue.(type) {
		case int64:
			sum += v
		default:
			return 0, true, fmt.Errorf("unable to parse the enum-identifier '%s' as an integer (received type: %T)", ref, v)
		}
	}

	return sum, true, nil
}

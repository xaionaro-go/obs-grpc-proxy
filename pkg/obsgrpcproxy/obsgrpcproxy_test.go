package obsgrpcproxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAbstractObject(t *testing.T) {
	type structType struct {
		A string
		B string
	}
	obj := structType{
		A: "some string",
		B: "another string",
	}
	abstractObj := ToAbstractObject(obj)

	m, err := FromAbstractObject[map[string]string](abstractObj)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"A": "some string",
		"B": "another string",
	}, m)
}

// +build integration_tests

package integrationtests

import (
	"testing"

	"github.com/gira-games/client/pkg/client"

	"github.com/stretchr/testify/require"

	// to register PostgreSQL driver
	_ "github.com/lib/pq"
)

func setup(t *testing.T) *client.Client {
	cl, err := client.New("http://localhost:21666")
	require.NoError(t, err)

	return cl
}

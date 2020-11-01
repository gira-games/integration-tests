// +build integration_tests

package integrationtests

import (
	"testing"

	"github.com/gira-games/client/pkg/client"

	"github.com/stretchr/testify/require"
)

// TestCreateAndGetAll creates a user, logs in, then creates two games
// and fetches them.
func TestCreateAndGetAll(t *testing.T) {
	cl := setup(t)

	user, err := cl.CreateUser(&client.CreateUserRequest{
		Email:    "games@test.com",
		Password: "password",
	})
	require.NoError(t, err)

	loginResp, err := cl.LoginUser(&client.LoginUserRequest{
		Email:    user.Email,
		Password: "password",
	})
	require.NoError(t, err)

	token := loginResp.Token

	batmanGame := createGame(t, cl, "Batman", token)
	acGame := createGame(t, cl, "AC", token)

	res, err := cl.GetGames(&client.GetGamesRequest{Token: token})
	require.NoError(t, err)

	require.Equal(t, 2, len(res.Games))
	require.Contains(t, res.Games, batmanGame)
	require.Contains(t, res.Games, acGame)
}

func createGame(t *testing.T, cl *client.Client, name, token string) *client.Game {
	res, err := cl.CreateGame(&client.CreateGameRequest{
		Token: token,
		Game: &client.Game{
			Name: name,
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Game.ID)
	require.Empty(t, res.Game.FranshiseID)
	require.Equal(t, name, res.Game.Name)

	return res.Game
}

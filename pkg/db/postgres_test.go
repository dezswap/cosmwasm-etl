package db

import (
	"testing"

	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/stretchr/testify/require"
)

func TestPostgresDb_Init(t *testing.T) {
	c := configs.New()

	pdb := PostgresDb{}
	require.NoError(t, pdb.Init(c.Rdb))
	defer pdb.Close()

	require.NoError(t, pdb.Db.Ping())
}

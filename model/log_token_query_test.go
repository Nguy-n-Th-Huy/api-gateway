package model

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// testGetLogsByTokenIdPaginated exercises the key-scoped log query every
// surface that reports one key's log uses (the public /key page and the
// Telegram bot integration) on one database: it must count only that key's
// rows, page them newest first, and still apply the user-visibility
// formatting that clears the upstream channel name.
func testGetLogsByTokenIdPaginated(t *testing.T, database *gorm.DB, logType common.DatabaseType) {
	t.Helper()

	previousLogDB := LOG_DB
	previousMainType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	LOG_DB = database
	common.SetDatabaseTypes(logType, logType)
	t.Cleanup(func() {
		LOG_DB = previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
	})

	if database.Migrator().HasTable("logs") {
		t.Skip("refusing to run the key-scoped log query test because a logs table already exists in this database")
	}
	require.NoError(t, database.AutoMigrate(&Log{}))
	t.Cleanup(func() { _ = database.Migrator().DropTable("logs") })

	seedLog := func(tokenId int, createdAt int64, modelName string) {
		t.Helper()
		require.NoError(t, database.Create(&Log{
			UserId:      1,
			CreatedAt:   createdAt,
			Type:        LogTypeConsume,
			TokenId:     tokenId,
			ModelName:   modelName,
			ChannelId:   42,
			ChannelName: "upstream-channel",
		}).Error)
	}
	seedLog(7, 100, "old-model")
	seedLog(7, 200, "middle-model")
	seedLog(7, 300, "new-model")
	seedLog(8, 400, "other-keys-model")

	firstPage, total, err := GetLogsByTokenIdPaginated(7, 0, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total, "the total counts only the requested key's rows")
	require.Len(t, firstPage, 2)
	assert.Equal(t, "new-model", firstPage[0].ModelName, "the newest row comes first")
	assert.Equal(t, "middle-model", firstPage[1].ModelName)
	for _, log := range firstPage {
		assert.Equal(t, 7, log.TokenId)
		assert.Empty(t, log.ChannelName, "the user-facing formatter must clear the upstream channel name")
	}

	secondPage, total, err := GetLogsByTokenIdPaginated(7, 2, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, secondPage, 1)
	assert.Equal(t, "old-model", secondPage[0].ModelName)

	emptyPage, total, err := GetLogsByTokenIdPaginated(9, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
	assert.Empty(t, emptyPage)
}

func TestGetLogsByTokenIdPaginatedSQLite(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testGetLogsByTokenIdPaginated(t, database, common.DatabaseTypeSQLite)
}

func TestGetLogsByTokenIdPaginatedMySQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not configured")
	}

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testGetLogsByTokenIdPaginated(t, database, common.DatabaseTypeMySQL)
}

func TestGetLogsByTokenIdPaginatedPostgreSQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not configured")
	}

	database, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testGetLogsByTokenIdPaginated(t, database, common.DatabaseTypePostgreSQL)
}
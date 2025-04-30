package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEnvTestFile(t *testing.T) {
	// Skip if file already exists
	if _, err := os.Stat(".env.test"); err == nil {
		t.Skip(".env.test file already exists")
	}

	// Create .env.test file with test credentials
	content := `TEST_DB_HOST=localhost
TEST_DB_USER=root
TEST_DB_PASS=1234
`
	err := os.WriteFile(".env.test", []byte(content), 0644)
	assert.NoError(t, err)
}

package dsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMessages_HappyPath(t *testing.T) {
	csvContent := `message_id,user_message,category,expected_tokens
1,"Hello, what services do you offer?",general,15
2,"Can you help me with pricing details?",pricing,20
3,"Thank you, goodbye!",closing,10
`
	tmpFile := filepath.Join(t.TempDir(), "messages.csv")
	require.NoError(t, os.WriteFile(tmpFile, []byte(csvContent), 0644))

	messages, err := LoadMessages(tmpFile)

	require.NoError(t, err)
	require.Len(t, messages, 3)

	assert.Equal(t, "1", messages[0].ID)
	assert.Equal(t, "Hello, what services do you offer?", messages[0].Text)
	assert.Equal(t, "general", messages[0].Category)
	assert.Equal(t, 15, messages[0].ExpectedTokens)

	assert.Equal(t, "2", messages[1].ID)
	assert.Equal(t, "Can you help me with pricing details?", messages[1].Text)
	assert.Equal(t, "pricing", messages[1].Category)
	assert.Equal(t, 20, messages[1].ExpectedTokens)

	assert.Equal(t, "3", messages[2].ID)
	assert.Equal(t, "Thank you, goodbye!", messages[2].Text)
	assert.Equal(t, "closing", messages[2].Category)
	assert.Equal(t, 10, messages[2].ExpectedTokens)
}

func TestLoadMessages_MissingFile(t *testing.T) {
	_, err := LoadMessages("/nonexistent/path/messages.csv")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open messages file")
}

func TestLoadMessages_EmptyFile(t *testing.T) {
	csvContent := `message_id,user_message,category,expected_tokens
`
	tmpFile := filepath.Join(t.TempDir(), "empty.csv")
	require.NoError(t, os.WriteFile(tmpFile, []byte(csvContent), 0644))

	messages, err := LoadMessages(tmpFile)

	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestLoadMessages_SkipsEmptyMessages(t *testing.T) {
	csvContent := `message_id,user_message,category,expected_tokens
1,"Valid message",general,15
2,,general,10
3,"Another valid",pricing,20
`
	tmpFile := filepath.Join(t.TempDir(), "gaps.csv")
	require.NoError(t, os.WriteFile(tmpFile, []byte(csvContent), 0644))

	messages, err := LoadMessages(tmpFile)

	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "Valid message", messages[0].Text)
	assert.Equal(t, "Another valid", messages[1].Text)
}

func TestLoadMessages_DefaultsTokensToFifteen(t *testing.T) {
	csvContent := `message_id,user_message,category,expected_tokens
1,"Hello",general,
`
	tmpFile := filepath.Join(t.TempDir(), "no_tokens.csv")
	require.NoError(t, os.WriteFile(tmpFile, []byte(csvContent), 0644))

	messages, err := LoadMessages(tmpFile)

	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, 15, messages[0].ExpectedTokens)
}

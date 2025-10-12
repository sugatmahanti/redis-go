package main

import (
	"testing"
	"time"
)

func TestExtractRESPString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "PING command",
			input:    "*1\r\n$4\r\nPING\r\n",
			expected: []string{"PING"},
		},
		{
			name:     "ECHO command with argument",
			input:    "*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n",
			expected: []string{"ECHO", "hello"},
		},
		{
			name:     "GET command",
			input:    "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
			expected: []string{"GET", "key"},
		},
		{
			name:     "SET command without expiry",
			input:    "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n",
			expected: []string{"SET", "key", "value"},
		},
		{
			name:     "SET command with PX expiry",
			input:    "*5\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n$2\r\nPX\r\n$3\r\n100\r\n",
			expected: []string{"SET", "key", "value", "PX", "100"},
		},
		{
			name:     "empty or invalid input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "non-array input",
			input:    "+OK\r\n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRESPString(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("at index %d: expected %q, got %q", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestRedisMapSetValue(t *testing.T) {
	rm := make(RedisMap)

	rm.setValue("key1", "value1")

	val, found := rm.getValue("key1")
	if !found {
		t.Error("expected key1 to be found")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}
}

func TestRedisMapGetValueNonExistent(t *testing.T) {
	rm := make(RedisMap)

	val, found := rm.getValue("nonexistent")
	if found {
		t.Error("expected key to not be found")
	}
	if val != "" {
		t.Errorf("expected empty string, got %s", val)
	}
}

func TestRedisMapSetValueWithExpiry(t *testing.T) {
	rm := make(RedisMap)

	t.Run("PX expiry in milliseconds", func(t *testing.T) {
		rm.setValueWithExpiry("key1", "value1", "PX", 100)

		val, found := rm.getValue("key1")
		if !found {
			t.Error("expected key1 to be found")
		}
		if val != "value1" {
			t.Errorf("expected value1, got %s", val)
		}
	})

	t.Run("EX expiry in seconds", func(t *testing.T) {
		rm.setValueWithExpiry("key2", "value2", "EX", 1)

		val, found := rm.getValue("key2")
		if !found {
			t.Error("expected key2 to be found")
		}
		if val != "value2" {
			t.Errorf("expected value2, got %s", val)
		}
	})
}

func TestRedisMapExpiry(t *testing.T) {
	rm := make(RedisMap)

	t.Run("expired key should be deleted", func(t *testing.T) {
		rm.setValueWithExpiry("expiring", "value", "PX", 50)

		val, found := rm.getValue("expiring")
		if !found {
			t.Error("expected key to be found immediately after setting")
		}
		if val != "value" {
			t.Errorf("expected value, got %s", val)
		}

		time.Sleep(100 * time.Millisecond)

		val, found = rm.getValue("expiring")
		if found {
			t.Error("expected key to be expired and not found")
		}
		if val != "" {
			t.Errorf("expected empty string for expired key, got %s", val)
		}

		if _, exists := rm["expiring"]; exists {
			t.Error("expected expired key to be deleted from map")
		}
	})

	t.Run("non-expired key should still be accessible", func(t *testing.T) {
		rm.setValueWithExpiry("notexpired", "value", "PX", 500)

		time.Sleep(100 * time.Millisecond)

		val, found := rm.getValue("notexpired")
		if !found {
			t.Error("expected key to still be found")
		}
		if val != "value" {
			t.Errorf("expected value, got %s", val)
		}
	})
}

func TestRedisMapOverwrite(t *testing.T) {
	rm := make(RedisMap)

	rm.setValue("key", "value1")
	rm.setValue("key", "value2")

	val, found := rm.getValue("key")
	if !found {
		t.Error("expected key to be found")
	}
	if val != "value2" {
		t.Errorf("expected value2, got %s", val)
	}
}

func TestRedisMapNilExpiry(t *testing.T) {
	rm := make(RedisMap)

	rm.setValue("key", "value")

	entry := rm["key"]
	if entry.Expiry != nil {
		t.Error("expected nil expiry for key without expiration")
	}

	val, found := rm.getValue("key")
	if !found {
		t.Error("expected key to be found")
	}
	if val != "value" {
		t.Errorf("expected value, got %s", val)
	}
}

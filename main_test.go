package main

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

func TestReadRESP(t *testing.T) {
	// Test 1: Parsing array with bulk strings
	resp := []byte("*2\r\n$4\r\nPING\r\n$4\r\nPONG\r\n")
	simulate := bufio.NewReader(bytes.NewReader(resp))
	expected := []string{"PING", "PONG"}
	result, err := ReadRESP(simulate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !equalSlices(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	// Test 2: Empty array parsing
	resp = []byte("*0\r\n")
	simulate = bufio.NewReader(bytes.NewReader(resp))
	expected = []string{}
	result, err = ReadRESP(simulate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !equalSlices(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	// Test 3: Bulk string with empty value
	resp = []byte("$0\r\n\r\n")
	simulate = bufio.NewReader(bytes.NewReader(resp))
	expected = []string{""}
	result, err = ReadRESP(simulate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !equalSlices(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	// Test 4: Incomplete message (should return error)
	resp = []byte("*2\r\n$4\r\nPING\r\n$4\r\nPO")
	simulate = bufio.NewReader(bytes.NewReader(resp))
	_, err = ReadRESP(simulate)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestStoreTypeHandling(t *testing.T) {
	store := NewStore()
	
	// Test 1: Set and get a string value
	result := store.Set("stringkey", "hello")
	if result != "OK" {
		t.Fatalf("expected OK, got %s", result)
	}
	
	value, exists, isCorrectType := store.Get("stringkey")
	if !exists || !isCorrectType || value != "hello" {
		t.Fatalf("expected (hello, true, true), got (%s, %v, %v)", value, exists, isCorrectType)
	}
	
	// Test 2: Create a non-string entry for testing WRONGTYPE
	store.SetForTesting("listkey", TypeList, []string{"item1", "item2"})
	
	// Try to GET a list key (should return wrong type)
	value, exists, isCorrectType = store.Get("listkey")
	if !exists || isCorrectType {
		t.Fatalf("expected (empty, true, false) for wrong type, got (%s, %v, %v)", value, exists, isCorrectType)
	}
	
	// Test 3: Check key type helper
	keyType, exists := store.KeyType("stringkey")
	if !exists || keyType != TypeString {
		t.Fatalf("expected (string, true), got (%s, %v)", keyType, exists)
	}
	
	keyType, exists = store.KeyType("listkey")
	if !exists || keyType != TypeList {
		t.Fatalf("expected (list, true), got (%s, %v)", keyType, exists)
	}
	
	keyType, exists = store.KeyType("nonexistent")
	if exists {
		t.Fatalf("expected (empty, false), got (%s, %v)", keyType, exists)
	}
	
	// Test 4: Delete operations
	count := store.Del("stringkey", "listkey", "nonexistent")
	if count != 2 {
		t.Fatalf("expected 2 deleted keys, got %d", count)
	}
}

func TestStoreConcurrency(t *testing.T) {
	store := NewStore()
	
	// Set initial values
	store.Set("concurrent1", "value1")
	store.Set("concurrent2", "value2")
	
	// Test concurrent reads (should not block each other)
	done := make(chan bool, 10)
	
	// Start 5 concurrent readers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				value, exists, isCorrectType := store.Get("concurrent1")
				if !exists || !isCorrectType || value != "value1" {
					t.Errorf("Reader %d: expected (value1, true, true), got (%s, %v, %v)", id, value, exists, isCorrectType)
				}
				
				value, exists, isCorrectType = store.Get("concurrent2")
				if !exists || !isCorrectType || value != "value2" {
					t.Errorf("Reader %d: expected (value2, true, true), got (%s, %v, %v)", id, value, exists, isCorrectType)
				}
			}
			done <- true
		}(i)
	}
	
	// Start 2 concurrent writers
	for i := 0; i < 2; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				key := fmt.Sprintf("writer_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				result := store.Set(key, value)
				if result != "OK" {
					t.Errorf("Writer %d: expected OK, got %s", id, result)
				}
				
				// Verify the written value
				readValue, exists, isCorrectType := store.Get(key)
				if !exists || !isCorrectType || readValue != value {
					t.Errorf("Writer %d: expected (%s, true, true), got (%s, %v, %v)", id, value, readValue, exists, isCorrectType)
				}
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 7; i++ {
		<-done
	}
	
	// Verify final state
	value, exists, isCorrectType := store.Get("concurrent1")
	if !exists || !isCorrectType || value != "value1" {
		t.Fatalf("Final state: expected (value1, true, true), got (%s, %v, %v)", value, exists, isCorrectType)
	}
}

func TestStoreBasicOperations(t *testing.T) {
	store := NewStore()
	
	// Test 1: Set and get a value
	result := store.Set("foo", "bar")
	if result != "OK" {
		t.Fatalf("expected OK, got %s", result)
	}
	
	value, exists, isCorrectType := store.Get("foo")
	if !exists || !isCorrectType || value != "bar" {
		t.Fatalf("expected (bar, true, true), got (%s, %v, %v)", value, exists, isCorrectType)
	}
	
	// Test 2: Get non-existent key
	value, exists, isCorrectType = store.Get("nosuch")
	if exists || !isCorrectType {
		t.Fatalf("expected (empty, false, true) for non-existent key, got (%s, %v, %v)", value, exists, isCorrectType)
	}
	
	// Test 3: Replace existing value
	store.Set("foo", "newvalue")
	value, exists, isCorrectType = store.Get("foo")
	if !exists || !isCorrectType || value != "newvalue" {
		t.Fatalf("expected (newvalue, true, true), got (%s, %v, %v)", value, exists, isCorrectType)
	}
	
	// Test 4: Delete existing key
	count := store.Del("foo")
	if count != 1 {
		t.Fatalf("expected 1 deleted key, got %d", count)
	}
	
	// Test 5: Get after deletion
	value, exists, isCorrectType = store.Get("foo")
	if exists {
		t.Fatalf("expected key to be deleted, but found (%s, %v, %v)", value, exists, isCorrectType)
	}
	
	// Test 6: Delete non-existent key
	count = store.Del("nonexistent")
	if count != 0 {
		t.Fatalf("expected 0 deleted keys, got %d", count)
	}
	
	// Test 7: Delete multiple keys
	store.Set("key1", "value1")
	store.Set("key2", "value2")
	store.Set("key3", "value3")
	
	count = store.Del("key1", "key2", "nonexistent", "key3")
	if count != 3 {
		t.Fatalf("expected 3 deleted keys, got %d", count)
	}
	
	// Verify all keys are gone
	_, exists, _ = store.Get("key1")
	if exists {
		t.Fatalf("expected key1 to be deleted")
	}
	_, exists, _ = store.Get("key2")
	if exists {
		t.Fatalf("expected key2 to be deleted")
	}
	_, exists, _ = store.Get("key3")
	if exists {
		t.Fatalf("expected key3 to be deleted")
	}
}


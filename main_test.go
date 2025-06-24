package main

import (
	"bufio"
	"bytes"
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


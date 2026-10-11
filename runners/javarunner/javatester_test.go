package javarunner

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestJavaListTests(t *testing.T) {
	dir := t.TempDir()

	testDir := filepath.Join(dir, "src", "test", "java")
	os.MkdirAll(testDir, 0755)

	os.WriteFile(filepath.Join(testDir, "MathTest.java"), []byte(""), 0644)

	tester := Javatester{
		Timeout: 60 * time.Second,
	}
	sawTestFileProgress := false
	tester.SetDiscoveryProgress(func(completed, total int, item string) {
		if total != -1 || item == "" {
			t.Errorf("unexpected discovery progress: total=%d item=%q", total, item)
		}
		if completed == 1 && filepath.Base(item) == "MathTest.java" {
			sawTestFileProgress = true
		}
	})
	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) != 1 || tests[0] != "MathTest" {
		t.Errorf("Expected MathTest, got %v", tests)
	}
	if !sawTestFileProgress {
		t.Error("discovery progress did not report the discovered Java test file")
	}
}

func TestJavaListTestsFindsCaseInsensitiveKotlinFilesAndDeclaredClasses(t *testing.T) {
	dir := t.TempDir()
	testDir := filepath.Join(dir, "src", "test", "kotlin", "com", "example")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(testDir, "test.kt")
	testSource := []byte("package com.example\n\nclass AppTest {\n    fun testGreeting() {}\n}\n")
	if err := os.WriteFile(testFile, testSource, 0644); err != nil {
		t.Fatal(err)
	}

	tester := Javatester{Timeout: time.Minute}
	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tests, []string{"com.example.AppTest"}) {
		t.Fatalf("ListTests() = %v, want [com.example.AppTest]", tests)
	}
}

func TestJavaListTestsFindsLowercaseTestSuffix(t *testing.T) {
	dir := t.TempDir()
	testDir := filepath.Join(dir, "src", "test", "kotlin")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "mathTest.kt"), []byte("class MathTest"), 0644); err != nil {
		t.Fatal(err)
	}

	tester := Javatester{Timeout: time.Minute}
	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(tests)
	if !reflect.DeepEqual(tests, []string{"MathTest"}) {
		t.Fatalf("ListTests() = %v, want [MathTest]", tests)
	}
}

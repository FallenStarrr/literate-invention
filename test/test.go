package mytest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockDB struct {
	t *testing.T
}


func (m *MockDB) Begin() (*sql.Tx, error) {
	return nil, nil
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if query == "SELECT name, code, value FROM r_currency WHERE a_date = $1" {
		rows := mockRows(m.t)
		return rows, nil
	} else if query == "SELECT name, code, value FROM r_currency WHERE a_date = $1 AND code = $2" {
		rows := mockRows(m.t)
		return rows, nil
	}
	return nil, fmt.Errorf("unexpected query: %s", query)
}

func mockRows(t *testing.T) *sql.Rows {
	columns := []string{"name", "code", "value"}
	values := [][]interface{}{{"USD", "USD", 1.0}, {"EUR", "EUR", 1.2}}
	rows := sql.Rows{}
	rows.Columns = columns
	for _, v := range values {
		rows.Rows = append(rows.Rows, v)
	}
	return &rows
}

func TestSaveCurrencyHandler(t *testing.T) {
	// Given
	db := &MockDB{t}
	r := mux.NewRouter()
	r.HandleFunc("/currency/save/{date}", saveCurrencyHandler(db)).Methods("GET")
	ts := httptest.NewServer(r)
	defer ts.Close()

	date := time.Now().Format("2006-01-02")

	// When
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/currency/save/%s", ts.URL, date), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetCurrencyHandler(t *testing.T) {
	// Given
	db := &MockDB{t}
	r := mux.NewRouter()
	r.HandleFunc("/currency/{date}/{code}", getCurrencyHandler(db)).Methods("GET")
	ts := httptest.NewServer(r)
	defer ts.Close()

	date := time.Now().Format("2006-01-02")
	expectedResult := []map[string]interface{}{
		{"name": "USD", "code": "USD", "value": 1.0},
		{"name": "EUR", "code": "EUR", "value": 1.2},
	}

	// When
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/currency/%s", ts.URL, date), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	var result []map[string]interface{}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(bodyBytes, &result)
	require.NoError(t, err)

	// Then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, expectedResult, result)
}

func TestMain(m *testing.M) {
	// Create a temporary config file for testing
	tempConfigFile, err := ioutil.TempFile("", "config.json")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tempConfigFile.Name())

	func TestSaveCurrency(t *testing.T) {
    // Create a mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return some sample XML data
        fmt.Fprintln(w, `<?xml version="1.0" encoding="utf-8"?>
        <rates>
            <date>2022-04-28</date>
            <item>
                <fullname>Australian dollar</fullname>
                <title>AUD</title>
                <description>1.3241</description>
            </item>
            <item>
                <fullname>Canadian dollar</fullname>
                <title>CAD</title>
                <description>1.2688</description>
            </item>
        </rates>`)
    }))
    defer server.Close()

    // Override the HTTP client to use the mock server
    httpGet = func(url string) (*http.Response, error) {
        return server.Client().Get(server.URL)
    }
    defer func() { httpGet = http.Get }()

    // Initialize the test database
    config := Config{
        DBHost:     "localhost",
        DBPort:     5432,
        DBName:     "test_db",
        DBUser:     "test_user",
        DBPassword: "test_password",
    }
    db := initDB(config)
    _, err := db.Exec("CREATE TABLE R_CURRENCY (NAME TEXT, CODE TEXT, VALUE FLOAT, A_DATE DATE)")
    if err != nil {
        t.Fatalf("Error creating test table: %v", err)
    }

    // Test saving currency for a valid date
    err = saveCurrency(db, "2022-04-28")
    if err != nil {
        t.Fatalf("Error saving currency: %v", err)
    }

    // Check that the currency was saved to the database
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM R_CURRENCY").Scan(&count)
    if err != nil {
        t.Fatalf("Error querying database: %v", err)
    }
    if count != 2 {
        t.Fatalf("Expected 2 rows in table, got %d", count)
    }

    // Test saving currency for an invalid date
    err = saveCurrency(db, "2022-02-31")
    if err == nil {
        t.Fatalf("Expected error saving currency")
    }
}

func TestGetCurrency(t *testing.T) {
  // Set up a test server
  ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, `{"success":true,"timestamp":1620814189,"base":"USD","date":"2021-05-12","rates":{"CAD":1.21312,"EUR":0.825862,"GBP":0.706618,"JPY":108.722364,"KRW":1124.407481,"USD":1}}`)
  }))
  defer ts.Close()

  // Set up the client with the test server URL
  client := NewClient(ts.URL)

  // Call the getCurrency function
  currency, err := client.getCurrency("USD")

  // Check if an error occurred
  if err != nil {
    t.Errorf("Unexpected error: %v", err)
  }

  // Check if the currency was returned correctly
  if currency != 1.0 {
    t.Errorf("Expected currency rate for USD to be 1.0, but got %v", currency)
  }
}


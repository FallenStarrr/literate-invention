package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gorilla/mux"
    _ "github.com/lib/pq"
)

type Config struct {
    Port       string `json:"port"`
    DBHost     string `json:"host"`
    DBPort     int    `json:"db_port"`
    DBName     string `json:"name"`
    DBUser     string `json:"user"`
    DBPassword string `json:"password"`
}


type CurrencyRates struct {
	XMLName xml.Name   `xml:"rates"`
	Date    string     `xml:"date"`
	Items   []Currency `xml:"item"`
}

type Currency struct {
	Fullname    string  `xml:"fullname"`
	Title       string  `xml:"title"`
	Description float64 `xml:"description"`
}



func main() {
   
    config := loadConfig()
    db := initDB(config)
    r := mux.NewRouter()

 
    r.HandleFunc("/currency/save/{date}", saveCurrencyHandler(db)).Methods("GET")
    r.HandleFunc("/currency/{date}/{code}", getCurrencyHandler(db)).Methods("GET")

   
    log.Printf("Starting server on port %s\n", config.Port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", config.Port), r))
}

func loadConfig() Config {
    var config Config
    configFile, err := os.Open("config.json")
    if err != nil {
        log.Fatalf("Failed to open config file: %v", err)
    }
    jsonParser := json.NewDecoder(configFile)
    if err = jsonParser.Decode(&config); err != nil {
        log.Fatalf("Failed to parse config file: %v", err)
    }
    return config
}

func initDB(config Config) *sql.DB {
    connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        config.DBHost, config.DBPort, config.DBName, config.DBUser, config.DBPassword)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    return db
}

go func saveCurrency(date string) error {
	resp, err := http.Get(fmt.Sprintf("https://nationalbank.kz/rss/get_rates.cfm?fdate=%s", date))
	if err != nil {
			return err
	}
	defer resp.Body.Close()

	xmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
			return err
	}

	var currencyRates CurrencyRates
	err = xml.Unmarshal(xmlData, &currencyRates)
	if err != nil {
			return err
	}

	tx, err := db.Begin()
	if err != nil {
			return err
	}
	defer func() {
			if err := recover(); err != nil {
					tx.Rollback()
			}
	}()

	for _, item := range currencyRates..Items {
			currency := Currency{
					Name:  item.FullName,
					Code:  item.Title,
					Value: item.Description,
					Date:  date,
			}
			_, err = tx.Exec("INSERT INTO R_CURRENCY (NAME, CODE, VALUE, A_DATE) VALUES ($1, $2, $3, $4)", currency.Name, currency.Code, currency.Value, currency.Date)
			if err != nil {
					tx.Rollback()
					return err
			}
	}

	err = tx.Commit()
	if err != nil {
			return err
	}

	return nil
}


go func getCurrency(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	dateStr := params["date"]
	code := params["code"]

	// Parse the date parameter
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
	}

	var query string
	var args []interface{}
	if code == "" {
			// If code is not provided, select all currencies for the given date
			query = "SELECT name, code, value FROM r_currency WHERE a_date = $1"
			args = []interface{}{date}
	} else {
			// If code is provided, select the currency with the given code for the given date
			query = "SELECT name, code, value FROM r_currency WHERE a_date = $1 AND code = $2"
			args = []interface{}{date, code}
	}

	// Query the database
	rows, err := db.Query(query, args...)
	if err != nil {
			log.Printf("Error querying database: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
	}
	defer rows.Close()

	// Build the response
	var result []map[string]interface{}
	for rows.Next() {
			var name string
			var code string
			var value float64
			err = rows.Scan(&name, &code, &value)
			if err != nil {
					log.Printf("Error scanning row: %v", err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
			}
			result = append(result, map[string]interface{}{
					"name":  name,
					"code":  code,
					"value": value,
			})
	}

	// Write the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}


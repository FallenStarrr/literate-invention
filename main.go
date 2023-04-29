package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	_"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Config struct {
    Port       int `json:"port"`
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

  var config = loadConfig()
	 var db = initDB(config)


func main() {
   
	 
    r := mux.NewRouter()
 
    r.HandleFunc("/currency/save/{date}", saveCurrency).Methods("GET")
    r.HandleFunc("/currency/{date}/{code}", getCurrency).Methods("GET")

   
    log.Printf("Starting server on port %d\n", config.Port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r))
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
		fmt.Println(config)
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

 func saveCurrency(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date := params["date"]
	resp, err := http.Get(fmt.Sprintf("https://nationalbank.kz/rss/get_rates.cfm?fdate=%s", date))
	
	if err != nil {
			fmt.Print(err)
	}
	defer resp.Body.Close()

	xmlData, err := ioutil.ReadAll(resp.Body)
	// fmt.Println("XML", xmlData)
	if err != nil {
			fmt.Print(err)
	}

	var currencyRates CurrencyRates
	err = xml.Unmarshal(xmlData, &currencyRates)
	// fmt.Print(currencyRates)
	if err != nil {
			fmt.Print(err)
	}

	

 connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
        config.DBHost, config.DBPort, config.DBName, config.DBUser, config.DBPassword)
fmt.Println(connStr)
	for _, item := range currencyRates.Items {
			currency := Currency{
				  Fullname:  item.Fullname,
					Title :  item.Title,
					Description: item.Description,
			}
			_, err = db.Exec("INSERT INTO test (title, code, value) VALUES ($1, $2, $3)", currency.Fullname, currency.Title, currency.Description)
			if err != nil {
					fmt.Print(err)
			}
	}


}


 func getCurrency(w http.ResponseWriter, r *http.Request) {
	// params := mux.Vars(r)
	// dateStr := params["date"]
	// code := params["code"]

	// Parse the date parameter
	// date, err := time.Parse("15.04.2021", dateStr)
	// fmt.Println(date)
	// if err != nil {
	// 		http.Error(w, "Invalid date format", http.StatusBadRequest)
	// 		fmt.Print(err)
	// }

	var query string
	
	// if code == "" {
	// 		// If code is not provided, select all currencies for the given date
	// 		query = "SELECT * FROM test"
	// 		rows, err := db.Query(query, date)
	// 		fmt.Print(err)
	// 		defer rows.Close()
	// } else {
	// 		// If code is provided, select the currency with the given code for the given date
	// 		query = "SELECT name, code, value FROM r_currency WHERE a_date = $1 AND code = $2"
	// 		rows, err := db.Query(query, date, code)
	// 		fmt.Print(err)
	// 		defer rows.Close()
	// }


	    


	// Query the database
	query = "SELECT * FROM test"
	rows, err:= db.Query(query)
			
	
	if err != nil {
			log.Printf("Error querying database: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			fmt.Print(err) 
	}
	

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
					fmt.Print(err)
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


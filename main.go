package main

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"text/template"
)

type Connection struct {
	Vehicle           string
	DepartureTime     string
	DepartureLocation string
	ArrivalTime       string
	ArrivalLocation   string
}
type Connections struct {
	Connections []Connection
}

func getstops(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		isa := Connections{}
		isa.Connections = append(isa.Connections, Connection{
			Vehicle:           "Irgendein Bus",
			DepartureTime:     "Jetzt",
			ArrivalTime:       "Später",
			DepartureLocation: "Bielefeld",
			ArrivalLocation:   "Buxtehude"})
		tmpl := template.Must(template.ParseFiles("./index.html"))
		err := tmpl.Execute(w, isa)
		if err != nil {
			log.Default().Fatalln("Die Template will irgendwie nicht \\(-o-)/")
		}

	case http.MethodPost:

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Parser Error", http.StatusBadRequest)
		}

		stopOne := r.FormValue("origin")
		stopTwo := r.FormValue("destination")
		apiKey := os.Getenv("API_KEY")

		if apiKey == "" {

			fmt.Println("API key not set")
			return
		}
		if stopOne == "" || stopTwo == "" {
			stopOne = "Frankfurt Niebelungenplatz"
			stopTwo = "Frankfurt Hauptbahnhof"
		}

		originUrl := fmt.Sprintf("https://www.rmv.de/hapi/location.name?accessId=%s&input=%s&format=json", apiKey, url.QueryEscape(stopOne))
		destinationUrl := fmt.Sprintf("https://www.rmv.de/hapi/location.name?accessId=%s&input=%s&format=json", apiKey, url.QueryEscape(stopTwo))

		originJson := get_request(originUrl)
		destinationJson := get_request(destinationUrl)

		originID := get_id(originJson)
		destinationID := get_id(destinationJson)

		connectionURL := fmt.Sprintf("https://www.rmv.de/hapi/trip?accessId=%s&originId=%s&destId=%s&format=json",
			apiKey, url.QueryEscape(originID), url.QueryEscape(destinationID))
		connectionJson := get_request(connectionURL)
		final := get_connection(connectionJson)
		tmpl := template.Must(template.ParseFiles("./index.html"))
		err = tmpl.Execute(w, &final)
		if err != nil {
			log.Default().Fatalln("Die Template ist schon wieder ein Problemkind...", err)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func get_request(URL string) []byte {

	request, err := http.NewRequest("GET", URL, nil)

	if err != nil {
		log.Default().Fatalln("Fehler in get Request ", err)
	}

	request.Header.Set("User-Agent", "MyUserAgent/1.0")
	response, err := http.DefaultClient.Do(request)

	if err != nil {
		log.Default().Fatalln("Fehler in get Request Body Read", err)
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	if err != nil {
		log.Default().Fatalln("Fehler in get Request Body Read", err)
	}

	return body
}

func get_id(stop []byte) string {

	id, _, _, err := jsonparser.Get([]byte(stop), "stopLocationOrCoordLocation", "[0]", "StopLocation", "id")

	if err != nil {
		log.Panicln("Fehler beim Jsonparsen(StopLocationID)", err)
	}

	return string(id)
}

func get_connection(jsonString []byte) Connections {
	conns := Connections{}
	jsonparser.ArrayEach(jsonString, func(value []byte, _ jsonparser.ValueType, _ int, err error) {
		jsonparser.ArrayEach(value, func(result []byte, _ jsonparser.ValueType, _ int, err error) {
			connection := Connection{}
			if err != nil {
				log.Panicln("Fehler beim Jsonparsen(im Loop)", err)
			}

			Vehicle, _, _, err := jsonparser.Get(result, "name")
			connection.Vehicle = string(Vehicle)

			departureTime, _, _, err := jsonparser.Get(result, "Origin", "time")
			departureLocation, _, _, err := jsonparser.Get(result, "Freq", "journey", "[0]", "Stops", "Stop", "[0]", "name")
			connection.DepartureTime = string(departureTime)
			connection.DepartureLocation = string(departureLocation)

			arrivalTime, _, _, err := jsonparser.Get(result, "Destination", "time")
			arrivalLocation, _, _, err := jsonparser.Get(result, "Freq", "journey", "[0]", "Stops", "Stop", "[1]", "name")
			connection.ArrivalTime = string(arrivalTime)
			connection.ArrivalLocation = string(arrivalLocation)

			conns.Connections = append(conns.Connections, connection)

		}, "LegList", "Leg")
		conns.Connections = append(conns.Connections, Connection{
			Vehicle:           "-",
			DepartureTime:     "-",
			DepartureLocation: "_",
			ArrivalTime:       "-",
			ArrivalLocation:   "-",
		})
	}, "Trip")

	return conns
}

func main() {
	err := godotenv.Load()

	if err != nil {

		fmt.Println("Error loading .env file")

		return
	}

	http.HandleFunc("/", getstops)
	fmt.Println("Listening at http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

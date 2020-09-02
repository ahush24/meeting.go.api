package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

type Participant struct {
	Name  string `json:"Name,omitempty" bson:"Name,omitempty"`
	Email string `json:"Email,omitempty" bson:"Email,omitempty"`
	RSVP  string `json:"RSVP,omitempty" bson:"RSVP,omitempty"`
}

type Meeting struct {
	Title          string        `json:"title,omitempty" bson:"title,omitempty"`
	Participants   []Participant `bson:"participants"`
	Starttime      time.Time     `json:"Starttime,omitempty" bson:"Starttime,omitempty"`
	Endtime        time.Time     `json:"Endtime,omitempty" bson:"Endtime,omitempty"`
	Creation_Stamp time.Time     `json:"creation,omitempty" bson:"creation,omitempty"`
}

func addmeeting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		ct := r.Header.Get("content-type")
		if ct != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s'", ct)))
			return
		}
		var meet Meeting
		err = json.Unmarshal(bodyBytes, &meet)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		collection := client.Database("mylib").Collection("meet.api")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		meet.Creation_Stamp = time.Now()
		result, _ := collection.InsertOne(ctx, meet)
		json.NewEncoder(w).Encode(&result)
		json.NewEncoder(w).Encode(meet)
	} else {
		start_string := r.FormValue("start")
		end_string := r.FormValue("end")
		layout := "2006-01-02T15:04:05.000Z"
		starttime, err := time.Parse(layout, start_string)
		if err != nil {
			fmt.Println(err)
		}
		endtime, err := time.Parse(layout, end_string)
		if err != nil {
			fmt.Println(err)
		}
		// starttime, err := strconv.Atoi(s)
		// endtime, err := strconv.Atoi(e)
		collection := client.Database("mylib").Collection("meet.api")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		filterCursor, err := collection.Find(ctx, bson.D{
			{"Starttime", bson.D{
				{"$gt", starttime},
			}},
			{"Endtime", bson.D{
				{"$lt", endtime},
			}},
		})
		if err != nil {
			log.Fatal(err)
		}
		var meetingsFiltered []Meeting
		if err = filterCursor.All(ctx, &meetingsFiltered); err != nil {
			log.Fatal(err)
		}

		for _, meeting := range meetingsFiltered {
			fmt.Println(meeting)
		}
		json.NewEncoder(w).Encode(meetingsFiltered)
	}
}

func getmeetingbyid(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id := path.Base(r.URL.RequestURI())
		fmt.Println(id)
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			log.Fatal(err)
		}
		var meeting Meeting
		collection := client.Database("mylib").Collection("meet.api")
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		result := collection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&meeting)
		if result != nil {
			log.Fatal(err)
		}
		json.NewEncoder(w).Encode(meeting)
	} else {
		return
	}
}

func articles(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		email := r.URL.Query().Get("participant")
		fmt.Println(email)
		collection := client.Database("mylib").Collection("meet.api")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		filterCursor, err := collection.Find(ctx, bson.M{"participants.Email": email})
		if err != nil {
			log.Fatal(err)
		}
		var meetingsFiltered []Meeting
		if err = filterCursor.All(ctx, &meetingsFiltered); err != nil {
			log.Fatal(err)
		}
		fmt.Println(meetingsFiltered)
		json.NewEncoder(w).Encode(meetingsFiltered)
	} else {
		return
	}
}

func handlerequest() {
	http.HandleFunc("/meetings", addmeeting)
	http.HandleFunc("/meetings/", getmeetingbyid)
	http.HandleFunc("/articles", articles)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func startdbserver() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions := options.Client().ApplyURI("mongodb://127.0.0.1:27017/")
	client, _ = mongo.Connect(ctx, clientOptions)
}

func main() {
	fmt.Println("Starting the application...")
	startdbserver()
	handlerequest()
}

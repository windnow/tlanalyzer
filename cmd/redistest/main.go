package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-redis/redis"
)

type State struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Author struct {
	Name  string            `json:"nm"`
	State State             `json:"type"`
	Props map[string]string `json:"props"`
}

func main() {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	pong, err := client.Ping().Result()
	fmt.Println(pong, err)
	{
		json, err := json.Marshal(Author{
			Name:  "Yermek",
			Props: map[string]string{"A key": "A val", "B key": "B val"},
			State: State{
				Id:   1,
				Name: "Active",
			},
		})
		if err != nil {
			log.Fatal(err)
		}

		err = client.Set("ytzh", json, 0).Err()
		if err != nil {
			log.Fatal(err)
		}
	}
	val, err := client.Get("ytzh").Result()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(val)
	{
		json, _ := json.Marshal([]map[string]string{map[string]string{"MapKey1": "MapValue1", "MapKey2": "MapValue2"}, map[string]string{"MapKey1": "MapValue1", "MapKey2": "MapValue2"}})
		fmt.Println(string(json))
	}

}

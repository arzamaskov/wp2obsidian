package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Post struct {
	ID int `json:"id"`

	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
}

func main() {
	resp, err := http.Get("http://10.195.7.71/wiki/index.php?rest_route=/wp/v2/posts&per_page=100")
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}

	var posts []Post

	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		panic(err)
	}
	fmt.Println("posts:", len(posts))
	for _, post := range posts {
		fmt.Printf("%d\t%s\n", post.ID, post.Title.Rendered)
	}
}

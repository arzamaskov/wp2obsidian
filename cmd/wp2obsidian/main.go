package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Post struct {
	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
}

func main() {
	resp, err := http.Get("http://10.195.7.71/wiki/index.php?rest_route=/wp/v2/posts/113")
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var post Post

	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		panic(err)
	}

	fmt.Println(post.Title.Rendered)
}

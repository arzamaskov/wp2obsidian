package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Post struct {
	ID int `json:"id"`

	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
}

func main() {
	posts, err := fetchPosts("http://10.195.7.71/wiki")
	if err != nil {
		panic(err)
	}

	fmt.Println("posts: ", len(posts))

	for _, post := range posts {
		fmt.Printf("%d\t%s\n", post.ID, post.Title.Rendered)
	}
}

func fetchPosts(baseURL string) ([]Post, error) {
	var posts []Post

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/index.php?rest_route=/wp/v2/posts&per_page=100&page=%d", baseURL, page)

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		var pagePosts []Post

		err = json.NewDecoder(resp.Body).Decode(&pagePosts)
		resp.Body.Close()

		if err != nil {
			return nil, err
		}

		posts = append(posts, pagePosts...)

		totalPages, err := strconv.Atoi(resp.Header.Get("X-WP-TotalPages"))
		if err != nil {
			return nil, err
		}

		if page >= totalPages {
			break
		}
	}

	return posts, nil
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

type Post struct {
	ID int `json:"id"`

	Title struct {
		Rendered string `json:"rendered"`
	} `json:"title"`

	Content struct {
		Rendered string `json:"rendered"`
	} `json:"content"`

	Categories []int `json:"categories"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	baseURL := "http://10.195.7.71/wiki"

	categories, err := fetchCategories(baseURL)
	if err != nil {
		panic(err)
	}

	categoryNames := make(map[int]string, len(categories))

	for _, category := range categories {
		categoryNames[category.ID] = category.Name
	}

	posts, err := fetchPosts(baseURL)
	if err != nil {
		panic(err)
	}

	for _, post := range posts {
		if post.ID != 113 {
			continue
		}

		markdown, err := md.ConvertString(post.Content.Rendered)
		if err != nil {
			panic(err)
		}

		fmt.Println(markdown)

		fmt.Printf("%d\t%s\n", post.ID, post.Title.Rendered)
		for _, categoryID := range post.Categories {
			fmt.Printf(" - %s\n", categoryNames[categoryID])
		}
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

func fetchCategories(baseURL string) ([]Category, error) {
	url := fmt.Sprintf(
		"%s/index.php?rest_route=/wp/v2/categories&per_page=100",
		baseURL,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var categories []Category

	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func attachmentName(imageURL string) (string, error) {
	u, err := url.Parse(imageURL)
	if err != nil {
		return "", err
	}

	const prefix = "/wp-content/uploads/"

	idx := strings.Index(u.Path, prefix)
	if idx == -1 {
		return "", fmt.Errorf("unexpected image path: %s", u.Path)
	}

	path := strings.TrimPrefix(u.Path[idx:], prefix)
	name := strings.ReplaceAll(path, "/", "-")

	return name, nil
}

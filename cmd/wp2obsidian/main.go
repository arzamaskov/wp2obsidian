package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
)

type Post struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Modified string `json:"modified"`

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

	postFiles := make(map[int]string, len(posts))
	for _, post := range posts {
		title := html.UnescapeString(post.Title.Rendered)
		postFiles[post.ID] = sanitizeFilename(title) + ".md"
	}

	for _, post := range posts {
		html, err := localizeLinks(post.Content.Rendered, postFiles)
		if err != nil {
			fmt.Printf("post %d %q: localize links: %v\n", post.ID, post.Title.Rendered, err)
			continue
		}

		html, err = localizeImages(post.Content.Rendered, "output")
		if err != nil {
			fmt.Printf("post %d %q: localize images: %v\n", post.ID, post.Title.Rendered, err)
			continue
		}

		markdown, err := md.ConvertString(html)
		if err != nil {
			fmt.Printf("post %d %q: convert markdown: %v\n", post.ID, post.Title.Rendered, err)
			continue
		}

		if err := savePost(post, markdown, "output", categoryNames); err != nil {
			fmt.Printf("post %d %q: save: %v\n", post.ID, post.Title.Rendered, err)
			continue
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

func downloadImage(imageURL, outputDir string) (string, error) {
	name, err := attachmentName(imageURL)
	if err != nil {
		return "", err
	}

	attachmentsDir := filepath.Join(outputDir, "attachments")

	if err := os.MkdirAll(attachmentsDir, 0o755); err != nil {
		return "", err
	}

	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	path := filepath.Join(attachmentsDir, name)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}

	return filepath.Join("attachments", name), nil
}

func localizeImages(html, outputDir string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	var firstErr error

	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		if firstErr != nil {
			return
		}

		src, ok := s.Attr("src")
		if !ok {
			return
		}

		localPath, err := downloadImage(src, outputDir)
		if err != nil {
			firstErr = err
			return
		}

		s.SetAttr("src", localPath)

		s.RemoveAttr("srcset")
		s.RemoveAttr("sizes")
	})

	if firstErr != nil {
		return "", firstErr
	}

	result, err := doc.Find("body").Html()
	if err != nil {
		return "", err
	}

	return result, nil
}

func savePost(
	post Post,
	markdown, outputDir string,
	categoryNames map[int]string,
) error {
	articleDir := filepath.Join(outputDir, "articles")

	if err := os.MkdirAll(articleDir, 0o755); err != nil {
		return err
	}

	filename := sanitizeFilename(post.Title.Rendered) + ".md"
	path := filepath.Join(articleDir, filename)
	content := buildFrontmatter(post, categoryNames) + markdown

	return os.WriteFile(path, []byte(content), 0o644)
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
	)

	return strings.TrimSpace(replacer.Replace(name))
}

func buildFrontmatter(post Post, categoryNames map[int]string) string {
	var b strings.Builder

	b.WriteString("---\n")
	fmt.Fprintf(&b, "wordpress_id: %d\n", post.ID)
	fmt.Fprintf(&b, "created: %s\n", post.Date)
	fmt.Fprintf(&b, "modified: %s\n", post.Modified)

	if len(post.Categories) > 0 {
		b.WriteString("categories:\n")

		for _, id := range post.Categories {
			if name, ok := categoryNames[id]; ok {
				fmt.Fprintf(&b, "  - %q\n", name)
			}
		}
	}

	b.WriteString("---\n\n")

	return b.String()
}

func wordpressPostID(href string) (int, bool) {
	u, err := url.Parse(href)
	if err != nil {
		return 0, false
	}

	id := u.Query().Get("p")
	if id == "" {
		return 0, false
	}

	postID, err := strconv.Atoi(id)
	if err != nil {
		return 0, false
	}

	return postID, true
}

func localizeLinks(html string, postFiles map[int]string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}

		postID, ok := wordpressPostID(href)
		if !ok {
			return
		}

		filename, ok := postFiles[postID]
		if !ok {
			return
		}

		s.SetAttr("href", filename)
	})

	result, err := doc.Find("body").Html()
	if err != nil {
		return "", err
	}

	return result, nil
}

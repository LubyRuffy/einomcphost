package tools

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/LubyRuffy/einomcphost/internal/llm"
	"github.com/LubyRuffy/pagemd/pkg/pagecontent"
	"github.com/LubyRuffy/pageviewer"
	"github.com/cloudwego/eino/components/tool"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/pkg/errors"
)

type SearchRequest struct {
	Query         string `json:"query" jsonschema:"description=The query to search the web for"`
	Think         string `json:"think" jsonschema:"description=The reason why to search, what you have already known, and expect what"`
	NumResults    int    `json:"num_results,omitempty" jsonschema:"description=The result number need to return (default is 10)"`
	Debug         bool   `json:"debug,omitempty" jsonschema:"description=Enable debug mode to print detailed operation steps and results (default: false)"`
	VisitHomaPage bool   `json:"visit_home_page,omitempty" jsonschema:"description=Whether to visit the home page of the search engine (default: false)"`
}

// SearchResult represents a single search result from ddg
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// SearchResponse represents the response from a ddg search
type SearchResponse struct {
	Query string `json:"query"`
	//Text    string         `json:"text"`
	Results []*SearchResult `json:"results"`
}

// NewGoogleSearchLocal 创建一个本地复用用户浏览器执行Google搜索的工具
func NewGoogleSearchLocal() tool.BaseTool {
	return llm.MustTool(
		"google_search",
		"Search Google for information on a given query",
		GoogleSearch)
}

var browser *pageviewer.Browser

func getBrowser() *pageviewer.Browser {
	if browser != nil {
		return browser
	}
	var err error
	browser, err = pageviewer.NewBrowser(
		pageviewer.WithUserModeBrowser(true),
		pageviewer.WithChromePath("/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"),
	)
	if err != nil {
		panic(err)
	}

	return browser
}

func GoogleSearch(ctx context.Context, sr *SearchRequest) (*SearchResponse, error) {
	if sr.Debug {
		log.Printf("[DEBUG] Starting Google search with query: %s\n", sr.Query)
		log.Printf("[DEBUG] Number of results requested: %d\n", sr.NumResults)
	}

	if sr.Query == "" {
		return nil, fmt.Errorf("query is empty")
	}

	if sr.NumResults == 0 {
		sr.NumResults = 10
	}

	vo := pageviewer.NewVisitOptions(pageviewer.WithBrowser(getBrowser()))
	//defer browser.Close()

	if sr.Debug {
		log.Println("[DEBUG] Browser instance created successfully")
	}

	// Prepare response
	response := &SearchResponse{
		Query:   sr.Query,
		Results: []*SearchResult{},
	}

	url := "https://www.google.com/search?q=" + sr.Query
	if sr.VisitHomaPage {
		url = "https://www.google.com"
	}

	// Run the search
	err := browser.Run(url,
		func(page *rod.Page) error {
			if sr.VisitHomaPage {
				if sr.Debug {
					log.Println("[DEBUG] Navigated to Google homepage")
				}

				// Wait for the search box to appear
				searchBox := page.MustElement(`textarea[name="q"]`)
				if sr.Debug {
					log.Println("[DEBUG] Search box found")
				}

				// Clear any existing text and input the search query
				searchBox.MustSelectAllText().MustInput(sr.Query)
				if sr.Debug {
					log.Printf("[DEBUG] Entered search query: %s\n", sr.Query)
				}

				// Find and click the search button instead of pressing Enter
				// searchButton := page.MustElement(`input[name="btnK"]`)
				// if sr.Debug {
				// 	log.Println("[DEBUG] Search button found")
				// }
				// searchButton.MustClick()
				// if sr.Debug {
				// 	log.Println("[DEBUG] Search button clicked")
				// }
				page.Keyboard.MustType(input.Enter)

				// Wait for results to load
				err := browser.WaitPage(page, vo.PageOptions)
				if err != nil {
					return fmt.Errorf("failed to wait for page: %v", err)
				}
				//page.MustWaitStable().MustWaitIdle().MustWaitDOMStable()
				if sr.Debug {
					log.Println("[DEBUG] Page stabilized after search")
				}

				// Give a little extra time for all results to render
				time.Sleep(1 * time.Second)
				if sr.Debug {
					log.Println("[DEBUG] Additional wait time completed")
				}
			}

			// Extract search results
			results, err := page.Elements(`div.g`)
			if err != nil || len(results) == 0 {
				results = page.MustElements(`div[data-snc]`)
				if len(results) == 0 {
					if ele, err := page.Element(`form[id=captcha-form]`); err == nil && ele != nil {
						return errors.Wrap(err, "google shows captcha")
					}
					return fmt.Errorf("no search results found")
				}
			}

			if sr.Debug {
				log.Printf("[DEBUG] Found %d raw search results\n", len(results))
			}

			//response.Text = page.MustElement("body").MustText()

			// Process only the requested number of results
			count := 0
			for i, result := range results {
				if count >= sr.NumResults {
					break
				}

				if sr.Debug {
					log.Printf("[DEBUG] Processing result #%d\n", i+1)
				}

				// Try to extract the title, URL, and description
				var title, url, description string

				// Extract title and URL
				titleElem := result.MustElement("h3")
				if titleElem != nil {
					title = titleElem.MustText()

					// Find the parent <a> element to get the URL
					linkElem := titleElem.MustParent()
					href, err := linkElem.Attribute("href")
					if err == nil && href != nil {
						url = *href
					}
				}

				descElem, err := result.Element(`div[data-sncf="1"]`)
				if err != nil || descElem == nil {
					// Extract description
					descElem, err = result.Element("div > div > div.g > div > div > div:last-child")
				}
				if descElem != nil {
					description = descElem.MustText()
				}

				// Only add results with both title and URL
				if title != "" && url != "" {
					response.Results = append(response.Results, &SearchResult{
						Title:       title,
						URL:         url,
						Description: description,
					})
					count++

					if sr.Debug {
						log.Printf("[DEBUG] Added result #%d: %s\n", count, title)
						log.Printf("[DEBUG]   URL: %s\n", url)
						if description != "" {
							log.Printf("[DEBUG]   Description: %s\n", description)
						}
					}
				} else if sr.Debug {
					log.Println("[DEBUG] Skipped result due to missing title or URL")
				}
			}

			return nil
		}, vo.PageOptions)

	if err != nil {
		if sr.Debug {
			log.Printf("[DEBUG] Search failed with error: %v\n", err)
		}
		return nil, fmt.Errorf("search failed: %v", err)
	}

	if sr.Debug {
		log.Printf("[DEBUG] Search completed successfully with %d results\n", len(response.Results))
	}

	return response, nil
}

type FetchUrlRequest struct {
	URL string `json:"url" jsonschema:"description=The URL to fetch the content of"`
}

type FetchUrlResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func FetchUrl(ctx context.Context, fr *FetchUrlRequest) (*FetchUrlResponse, error) {
	ci, err := pagecontent.NewAnalysis(
		pagecontent.WithURL(fr.URL),
		pagecontent.WithOnMainContentFound(func(s string) {
			log.Println("found main node, size:", len(s))
		}),
		pagecontent.WithOnHtmlFetched(func(htmlContent string) {
			log.Println("fetched html, size:", len(htmlContent))
		}),
	).ExtractMainContent()
	if err != nil {
		log.Fatal(err)
	}

	return &FetchUrlResponse{
		Title:   ci.Title,
		Content: ci.Markdown,
	}, nil
}

// NewFetchUrlLocal 调用本地浏览器完成网页内容的获取
func NewFetchUrlLocal() tool.BaseTool {
	return llm.MustTool(
		"fetch_url",
		"Fetch the content of a given URL",
		FetchUrl,
	)
}

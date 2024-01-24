package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	
	"net/http"
	"strings"

	
	"golang.org/x/time/rate"
)

var ErrBlocked = errors.New("duckduckgo block")

var RateLimit = rate.NewLimiter(rate.Inf, 0)

type Result struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

var DuckDuckGoDomains = map[string]string{
	"us": "https://duckduckgo.com/?t=h_&q=",
}

type SearchOptions struct {
	CountryCode  string
	LanguageCode string
	Limit        int
	Start        int
	ProxyAddr    string
}

func Search(ctx context.Context, searchTerm string, opts ...SearchOptions) ([]Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	buildDuckDuckGoURL := func(searchTerm string) string {
		searchTerm = strings.ReplaceAll(searchTerm, " ", "+")
		return fmt.Sprintf("https://duckduckgo.com/html/?q=%s", searchTerm)
	}

	url := buildDuckDuckGoURL(searchTerm)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrBlocked
	}

	var searchResponse struct {
		Results []Result `json:"Results"`
	}
	err = json.NewDecoder(resp.Body).Decode(&searchResponse)
	if err != nil {
		return nil, err
	}

	return searchResponse.Results, nil
}

func url(searchTerm string, countryCode string, languageCode string, limit int, start int) string {
	searchTerm = strings.Trim(searchTerm, " ")
	searchTerm = strings.Replace(searchTerm, " ", "+", -1)
	countryCode = strings.ToLower(countryCode)

	var url string

	if duckduckgoBase, found := DuckDuckGoDomains[countryCode]; found {
		if start == 0 {
			url = fmt.Sprintf("%s%s&hl=%s", duckduckgoBase, searchTerm, languageCode)
		} else {
			url = fmt.Sprintf("%s%s&hl=%s&start=%d", duckduckgoBase, searchTerm, languageCode, start)
		}
	} else {
		if start == 0 {
			url = fmt.Sprintf("%s%s&hl=%s", DuckDuckGoDomains["us"], searchTerm, languageCode)
		} else {
			url = fmt.Sprintf("%s%s&hl=%s&start=%d", DuckDuckGoDomains["us"], searchTerm, languageCode, start)
		}
	}

	if limit != 0 {
		url = fmt.Sprintf("%s&num=%d", url, limit)
	}

	return url
}

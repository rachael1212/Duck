package search

import (
        "context"
        "errors"
        "fmt"
        "github.com/gocolly/colly/v2"
        "github.com/gocolly/colly/v2/proxy"
        "golang.org/x/time/rate"
        "math/rand"
        "strings"
	"io/ioutil"
	"net/http"
)

var ErrBlocked = errors.New("duckduckgo block")

var RateLimit = rate.NewLimiter(rate.Inf, 0)

type Result struct {
        URL string `json:"url"`

        Title string `json:"title"`

        Description string `json:"description"`
}

type SearchOptions struct {
        CountryCode string

        LanguageCode string

        Limit int

        Start int

        //OverLimit bool

        ProxyAddr string
}

func DuckDuckGoDomains(searchTerm string) (string, error) {
	ctx := context.Background()

	searchTerm = strings.Trim(searchTerm, " ")
	searchTerm = strings.Replace(searchTerm, " ", "+", -1)

	// Construct the URL for the custom HTTP request to DuckDuckGo
	url := fmt.Sprintf("https://duckduckgo.com/html/?q=%s", searchTerm)

	// Make the HTTP request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	// Set any necessary headers or parameters
	req.Header.Set("User-Agent", "Your User-Agent")

	// Perform the request
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Return the response body as a string
	return string(body), nil
}

func Search(ctx context.Context, searchTerm string, opts ...SearchOptions) ([]Result, error) {

        if ctx == nil {
                ctx = context.Background()
        }

        if err := RateLimit.Wait(ctx); err != nil {
                return nil, err
        }

        c := colly.NewCollector(
                colly.AllowURLRevisit(),
                colly.MaxDepth(1),
        )

        if len(opts) == 0 {
                opts = append(opts, SearchOptions{})
        }

        var lc string
        if opts[0].LanguageCode == "" {
                lc = "en"
        } else {
                lc = opts[0].LanguageCode
        }

	// Perform the search using DuckDuckGo custom HTTP request
	results, err := DuckDuckGoDomains(searchTerm)
	if err != nil {
		return nil, err
	}

      
        c.OnRequest(func(r *colly.Request) {

                if opts[0].ProxyAddr != "" {
                        rp, err := proxy.RoundRobinProxySwitcher(opts[0].ProxyAddr)
                        if err != nil {

                        }
                        c.SetProxyFunc(rp)
                }
                r.Headers.Set("X-Requested-With", "XMLHttpRequest")
                //r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
                r.Headers.Set("accept-encoding", "null")
                //r.Headers.Set("Accept-Language", "it-IT,it;q=0.8,en-US;q=0.5,en;q=0.3,az")
                //r.Headers.Set("cache-control", "no-cache")
                //r.Headers.Set("Content-Type", "text/html; charset=UTF-8")
                //r.Headers.Set("Content-language", "en")
                r.Headers.Set("Cookie", "CONSENT=YES+srp.gws-20210802-0-RC1.us+FX+815; 1P_JAR=2021-08-05-10")

                //r.Headers.Set("Referer", "https://www.duckduckgo.com/")
                //r.Headers.Set("Sec-Fetch-Dest" , "document")
                //r.Headers.Set("Sec-Fetch-Site", "same-origin")
                //r.Headers.Set("Sec-Fetch-Mode", "navigate")
                //r.Headers.Set("TE", "trailers")
                //r.Headers.Set("upgrade-insecure-requests", "1")

                r.Headers.Set("User-Agent", uaGens[rand.Intn(len(uaGens))]())

                if err := ctx.Err(); err != nil {
                        r.Abort()
                        rErr = err
                        return
                }
        })

        c.OnResponse(func(r *colly.Response) {

                if strings.Contains(string(r.Body[:]), "did not match any documents.") {
                        rErr = errors.New("didn't find any results")
                }

                //log.Println(string(r.Body[:]))
                //log.Println("========================================")
        })

        c.OnError(func(r *colly.Response, err error) {
                rErr = err

        })

        c.OnHTML("div.g", func(e *colly.HTMLElement) {
                sel := e.DOM

                linkHref, _ := sel.Find("a").Attr("href")
                linkText := strings.TrimSpace(linkHref)
                titleText := strings.TrimSpace(sel.Find("div > div > a > h3").Text())

                descText := strings.TrimSpace(sel.Find("div > div > div > span > span").Text())

                if linkText != "" && linkText != "#" {
                        result := Result{
                                URL:         linkText,
                                Title:       titleText,
                                Description: descText,
                        }
                        results = append(results, result)
                }
        })

        limit := opts[0].Limit

        //if opts[0].OverLimit {
        limit = int(float64(opts[0].Limit) * 1.5)
        //}

        url := url(searchTerm, opts[0].CountryCode, lc, limit, opts[0].Start)

        if opts[0].ProxyAddr != "" {
                rp, err := proxy.RoundRobinProxySwitcher(opts[0].ProxyAddr)
                if err != nil {
                        return nil, err
                }
                c.SetProxyFunc(rp)
        }

        c.Visit(url)

        if rErr != nil {
                if strings.Contains(rErr.Error(), "Too Many Requests") {
                        return nil, ErrBlocked
                }
                return nil, rErr
        }

        if opts[0].Limit != 0 && len(results) > opts[0].Limit {
                return results[:opts[0].Limit], nil
        }

	// Return the search results
        return []string{results}, nil
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

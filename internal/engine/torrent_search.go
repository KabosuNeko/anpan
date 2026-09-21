package engine

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KabosuNeko/anpan/internal/units"
)

// TorrentSearchResult represents a single torrent release from an indexer.
type TorrentSearchResult struct {
	Title     string `json:"title"`
	InfoHash  string `json:"info_hash"`
	Magnet    string `json:"magnet"`
	SizeBytes int64  `json:"size_bytes"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
	Source    string `json:"source"`
	Category  string `json:"category"`
}

// SearchOptions holds parameters for searching torrents.
type SearchOptions struct {
	Category string // "all", "anime", "movies", "tv", "games"
	Limit    int
	SortBy   string // "seeds", "size", "name"
	Page     int    // 1-based page number
}

var searchHTTPClient = &http.Client{
	Timeout: 8 * time.Second,
}

const (
	uaAnpan   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) anpan/1.0.0"
	uaTorlink = "anpan/torlink-client"
)

// fetchHTML performs a GET request with searchHTTPClient and returns the response body.
func fetchHTML(ctx context.Context, endpoint string, userAgent string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := searchHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func parseFlexibleInt(val interface{}) int {
	switch v := val.(type) {
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func parseFlexibleInt64(val interface{}) int64 {
	switch v := val.(type) {
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
}

// SearchTorrents concurrently searches multiple open torrent sources and returns merged results.
func SearchTorrents(ctx context.Context, query string, opts *SearchOptions) ([]TorrentSearchResult, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}

	cat := strings.ToLower(strings.TrimSpace(opts.Category))
	if cat == "" {
		cat = "all"
	}

	page := opts.Page
	if page < 1 {
		page = 1
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allResults []TorrentSearchResult

	collect := func(fn func(context.Context, string, string, int) ([]TorrentSearchResult, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := fn(ctx, query, cat, page)
			if err == nil && len(res) > 0 {
				mu.Lock()
				allResults = append(allResults, res...)
				mu.Unlock()
			}
		}()
	}

	switch cat {
	case "anime":
		collect(searchSubsPlease)
		collect(searchNyaa)
		collect(searchPirateBay)
	case "movies":
		collect(searchYTS)
		collect(search1337x)
		collect(searchPirateBay)
	case "tv":
		collect(searchEZTV)
		collect(search1337x)
		collect(searchPirateBay)
	case "games":
		collect(searchFitGirl)
		collect(search1337x)
		collect(searchNyaa)
		collect(searchPirateBay)
	default: // "all"
		collect(searchSubsPlease)
		collect(searchNyaa)
		collect(searchYTS)
		collect(searchEZTV)
		collect(searchPirateBay)
		collect(search1337x)
		collect(searchFitGirl)
	}

	wg.Wait()

	seen := make(map[string]TorrentSearchResult)
	for _, r := range allResults {
		ih := strings.ToLower(r.InfoHash)
		if ih == "" {
			ih = ParseInfoHashFromMagnet(r.Magnet)
		}
		if ih == "" {
			ih = r.Title
		}

		existing, found := seen[ih]
		if !found || r.Seeders > existing.Seeders {
			seen[ih] = r
		}
	}

	var deduped []TorrentSearchResult
	for _, r := range seen {
		deduped = append(deduped, r)
	}

	SortSearchResults(deduped, opts.SortBy)

	if opts.Limit > 0 && len(deduped) > opts.Limit {
		deduped = deduped[:opts.Limit]
	}

	return deduped, nil
}

// SortSearchResults sorts torrent search results in place using the given sort mode.
func SortSearchResults(results []TorrentSearchResult, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "size", "size-desc":
		sort.Slice(results, func(i, j int) bool {
			return results[i].SizeBytes > results[j].SizeBytes
		})
	case "size-asc", "size-up", "smallest":
		sort.Slice(results, func(i, j int) bool {
			if results[i].SizeBytes <= 0 && results[j].SizeBytes > 0 {
				return false
			}
			if results[i].SizeBytes > 0 && results[j].SizeBytes <= 0 {
				return true
			}
			if results[i].SizeBytes == results[j].SizeBytes {
				return results[i].Seeders > results[j].Seeders
			}
			return results[i].SizeBytes < results[j].SizeBytes
		})
	case "peers", "leechers", "activity":
		sort.Slice(results, func(i, j int) bool {
			pi := results[i].Seeders + results[i].Leechers
			pj := results[j].Seeders + results[j].Leechers
			if pi == pj {
				return results[i].Seeders > results[j].Seeders
			}
			return pi > pj
		})
	case "source":
		sort.Slice(results, func(i, j int) bool {
			si := strings.ToLower(results[i].Source)
			sj := strings.ToLower(results[j].Source)
			if si == sj {
				return results[i].Seeders > results[j].Seeders
			}
			return si < sj
		})
	case "name":
		sort.Slice(results, func(i, j int) bool {
			return strings.ToLower(results[i].Title) < strings.ToLower(results[j].Title)
		})
	default: // "seeds"
		sort.Slice(results, func(i, j int) bool {
			if results[i].Seeders == results[j].Seeders {
				return results[i].SizeBytes > results[j].SizeBytes
			}
			return results[i].Seeders > results[j].Seeders
		})
	}
}

func searchSubsPlease(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	var endpoint string
	if strings.TrimSpace(query) == "" {
		endpoint = fmt.Sprintf("https://subsplease.org/api/?f=latest&tz=Asia/Tokyo&p=%d", page)
	} else {
		endpoint = fmt.Sprintf("https://subsplease.org/api/?f=search&tz=Asia/Tokyo&s=%s&p=%d", url.QueryEscape(query), page)
	}

	body, err := fetchHTML(ctx, endpoint, uaAnpan)
	if err != nil {
		return nil, fmt.Errorf("subsplease: %w", err)
	}

	var data map[string]struct {
		Time        string `json:"time"`
		ReleaseDate string `json:"release_date"`
		Show        string `json:"show"`
		Episode     string `json:"episode"`
		Downloads   []struct {
			Res    string `json:"res"`
			Magnet string `json:"magnet"`
		} `json:"downloads"`
	}

	if err := json.Unmarshal(body, &data); err != nil || len(data) == 0 {
		return nil, nil
	}

	var results []TorrentSearchResult
	for _, item := range data {
		for _, dl := range item.Downloads {
			if dl.Magnet == "" {
				continue
			}
			title := fmt.Sprintf("[SubsPlease] %s - %s (%sp)", item.Show, item.Episode, dl.Res)
			ih := ParseInfoHashFromMagnet(dl.Magnet)
			sizeBytes := parseSizeFromMagnet(dl.Magnet)
			if sizeBytes == 0 {
				sizeBytes = 1400 * 1024 * 1024 // fallback ~1.4GB
			}
			results = append(results, TorrentSearchResult{
				Title:     title,
				InfoHash:  ih,
				Magnet:    EnrichMagnetWithTrackers(dl.Magnet),
				SizeBytes: sizeBytes,
				Seeders:   100, // Active swarm
				Leechers:  10,
				Source:    "SubsPlease",
				Category:  "anime",
			})
		}
	}
	return results, nil
}

func parseSizeFromMagnet(mag string) int64 {
	u, err := url.Parse(mag)
	if err != nil {
		return 0
	}
	n, err := strconv.ParseInt(u.Query().Get("xl"), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

type nyaaRSS struct {
	Channel struct {
		Items []struct {
			Title      string `xml:"title"`
			Link       string `xml:"link"`
			Seeders    int    `xml:"seeders"`
			Leechers   int    `xml:"leechers"`
			SizeStr    string `xml:"size"`
			InfoHash   string `xml:"infoHash"`
			CategoryId string `xml:"categoryId"`
		} `xml:"item"`
	} `xml:"channel"`
}

func searchNyaa(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	cParam := ""
	if cat == "games" {
		cParam = "&c=6_2"
	}

	mirrors := []string{
		"https://nyaa.net/?page=rss&q=%s&p=%d&s=seeders&o=desc" + cParam,
		"https://nyaa.si/?page=rss&q=%s&p=%d&s=seeders&o=desc" + cParam,
	}

	var lastErr error
	for _, mirror := range mirrors {
		endpoint := fmt.Sprintf(mirror, url.QueryEscape(query), page)
		reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		data, err := fetchHTML(reqCtx, endpoint, uaAnpan)
		cancel()
		if err != nil {
			lastErr = fmt.Errorf("nyaa: %w", err)
			continue
		}

		var feed nyaaRSS
		if err := xml.Unmarshal(data, &feed); err != nil {
			lastErr = err
			continue
		}

		var results []TorrentSearchResult
		for _, item := range feed.Channel.Items {
			ih := strings.ToLower(strings.TrimSpace(item.InfoHash))
			if ih == "" && strings.HasPrefix(item.Link, "magnet:") {
				ih = ParseInfoHashFromMagnet(item.Link)
			}
			mag := item.Link
			if !strings.HasPrefix(mag, "magnet:") && ih != "" {
				mag = BuildMagnet(ih, item.Title, nil)
			}

			sizeBytes := units.ParseBytes(item.SizeStr)

			itemCat := "anime"
			if cat == "games" || item.CategoryId == "6_2" {
				itemCat = "games"
			}

			results = append(results, TorrentSearchResult{
				Title:     item.Title,
				InfoHash:  ih,
				Magnet:    EnrichMagnetWithTrackers(mag),
				SizeBytes: sizeBytes,
				Seeders:   item.Seeders,
				Leechers:  item.Leechers,
				Source:    "Nyaa",
				Category:  itemCat,
			})
		}
		return results, nil
	}
	return nil, lastErr
}

type ytsResponse struct {
	Data struct {
		MovieCount int `json:"movie_count"`
		Movies     []struct {
			Title    string `json:"title_long"`
			Year     int    `json:"year"`
			Torrents []struct {
				Hash      string `json:"hash"`
				Quality   string `json:"quality"`
				Type      string `json:"type"`
				Seeds     int    `json:"seeds"`
				Peers     int    `json:"peers"`
				SizeBytes int64  `json:"size_bytes"`
			} `json:"torrents"`
		} `json:"movies"`
	} `json:"data"`
}

func searchYTS(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	mirrors := []string{
		"https://movies-api.accel.li/api/v2/list_movies.json?limit=50&page=%d&query_term=%s",
		"https://yts.mx/api/v2/list_movies.json?limit=50&page=%d&query_term=%s",
	}

	var lastErr error
	for _, mirror := range mirrors {
		endpoint := fmt.Sprintf(mirror, page, url.QueryEscape(query))
		data, err := fetchHTML(ctx, endpoint, uaTorlink)
		if err != nil {
			lastErr = err
			continue
		}

		var ytsData ytsResponse
		if err := json.Unmarshal(data, &ytsData); err != nil || ytsData.Data.MovieCount == 0 {
			continue
		}

		var results []TorrentSearchResult
		for _, m := range ytsData.Data.Movies {
			for _, t := range m.Torrents {
				title := fmt.Sprintf("%s [%s %s]", m.Title, t.Quality, strings.ToUpper(t.Type))
				mag := BuildMagnet(t.Hash, title, nil)
				results = append(results, TorrentSearchResult{
					Title:     title,
					InfoHash:  strings.ToLower(t.Hash),
					Magnet:    mag,
					SizeBytes: t.SizeBytes,
					Seeders:   t.Seeds,
					Leechers:  t.Peers,
					Source:    "YTS",
					Category:  "movies",
				})
			}
		}
		return results, nil
	}
	return nil, lastErr
}

type eztvTorrentItem struct {
	Title     string      `json:"title"`
	Filename  string      `json:"filename"`
	Hash      string      `json:"hash"`
	MagnetURL string      `json:"magnet_url"`
	SizeBytes interface{} `json:"size_bytes"`
	Seeds     interface{} `json:"seeds"`
	Peers     interface{} `json:"peers"`
}

type eztvResponse struct {
	Torrents []eztvTorrentItem `json:"torrents"`
}

// containsAllTokens reports whether haystack contains every token.
func containsAllTokens(haystack string, tokens []string) bool {
	for _, tok := range tokens {
		if !strings.Contains(haystack, tok) {
			return false
		}
	}
	return true
}

func searchEZTV(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	endpoint := fmt.Sprintf("https://eztvx.to/api/get-torrents?limit=100&page=%d", page)
	dataBytes, err := fetchHTML(ctx, endpoint, uaTorlink)
	if err != nil {
		return nil, fmt.Errorf("eztv: %w", err)
	}

	var data eztvResponse
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return nil, err
	}

	qTokens := strings.Fields(strings.ToLower(query))

	var results []TorrentSearchResult
	for _, t := range data.Torrents {
		title := t.Title
		if title == "" {
			title = t.Filename
		}
		if !containsAllTokens(strings.ToLower(title+" "+t.Filename), qTokens) {
			continue
		}

		ih := strings.ToLower(t.Hash)
		mag := t.MagnetURL
		if mag == "" && ih != "" {
			mag = BuildMagnet(ih, title, nil)
		}
		results = append(results, TorrentSearchResult{
			Title:     title,
			InfoHash:  ih,
			Magnet:    EnrichMagnetWithTrackers(mag),
			SizeBytes: parseFlexibleInt64(t.SizeBytes),
			Seeders:   parseFlexibleInt(t.Seeds),
			Leechers:  parseFlexibleInt(t.Peers),
			Source:    "EZTV",
			Category:  "tv",
		})
	}
	return results, nil
}

type apibayItem struct {
	ID       interface{} `json:"id"`
	Name     string      `json:"name"`
	InfoHash string      `json:"info_hash"`
	Leechers interface{} `json:"leechers"`
	Seeders  interface{} `json:"seeders"`
	Size     interface{} `json:"size"`
	Category interface{} `json:"category"`
}

func searchPirateBay(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	apibayCat := map[string]string{
		"anime":  "205",
		"tv":     "208",
		"movies": "207",
		"games":  "400",
	}

	var endpoint string
	if strings.TrimSpace(query) == "" {
		code := apibayCat[cat]
		if code == "" {
			code = "all"
		}
		endpoint = fmt.Sprintf("https://apibay.org/precompiled/data_top100_%s.json", code)
	} else {
		catParam := ""
		if code := apibayCat[cat]; code != "" {
			catParam = "&cat=" + code
		}
		endpoint = fmt.Sprintf("https://apibay.org/q.php?q=%s%s", url.QueryEscape(query), catParam)
	}

	body, err := fetchHTML(ctx, endpoint, uaTorlink)
	if err != nil {
		return nil, fmt.Errorf("apibay: %w", err)
	}

	var items []apibayItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	var results []TorrentSearchResult
	for _, item := range items {
		idStr := fmt.Sprintf("%v", item.ID)
		if idStr == "0" || item.InfoHash == "" || item.InfoHash == "0000000000000000000000000000000000000000" {
			continue
		}

		seeds := parseFlexibleInt(item.Seeders)
		leechs := parseFlexibleInt(item.Leechers)
		sizeBytes := parseFlexibleInt64(item.Size)

		resCat := cat
		if resCat == "all" || resCat == "" {
			catNum := parseFlexibleInt(item.Category)
			switch {
			case catNum >= 400 && catNum < 500:
				resCat = "games"
			case catNum == 205:
				resCat = "anime"
			case catNum == 208 || (catNum >= 200 && catNum < 300 && strings.Contains(strings.ToLower(item.Name), "s0")):
				resCat = "tv"
			case catNum >= 200 && catNum < 300:
				resCat = "movies"
			}
		}

		mag := BuildMagnet(item.InfoHash, item.Name, nil)
		results = append(results, TorrentSearchResult{
			Title:     item.Name,
			InfoHash:  strings.ToLower(item.InfoHash),
			Magnet:    mag,
			SizeBytes: sizeBytes,
			Seeders:   seeds,
			Leechers:  leechs,
			Source:    "TPB",
			Category:  resCat,
		})
	}

	// Window-paginate results (50 per page) so deep pages get distinct results
	perPage := 50
	startIdx := (page - 1) * perPage
	if startIdx >= len(results) {
		return nil, nil
	}
	endIdx := startIdx + perPage
	if endIdx > len(results) {
		endIdx = len(results)
	}
	return results[startIdx:endIdx], nil
}

func searchFitGirl(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	q := strings.TrimSpace(query)
	var endpoint string
	if q == "" {
		if page <= 1 {
			endpoint = "https://fitgirl-repacks.site/feed/"
		} else {
			endpoint = fmt.Sprintf("https://fitgirl-repacks.site/feed/?paged=%d", page)
		}
	} else {
		if page <= 1 {
			endpoint = fmt.Sprintf("https://fitgirl-repacks.site/?s=%s&feed=rss2", url.QueryEscape(q))
		} else {
			endpoint = fmt.Sprintf("https://fitgirl-repacks.site/?s=%s&feed=rss2&paged=%d", url.QueryEscape(q), page)
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	bodyBytes, err := fetchHTML(reqCtx, endpoint, "torlink (+https://www.npmjs.com/package/torlnk)")
	if err != nil {
		return nil, nil // graceful fail-soft on DDoS-Guard / network timeouts
	}

	return parseFitGirlRSS(string(bodyBytes), q), nil
}

var fitgirlItemRegex = regexp.MustCompile(`(?s)<item>(.*?)</item>`)
var fitgirlTitleRegex = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
var fitgirlMagnetRegex = regexp.MustCompile(`magnet:\?xt=urn:btih:[a-zA-Z0-9]+[^\s"'<>]*`)

func parseFitGirlRSS(xmlContent, query string) []TorrentSearchResult {
	items := fitgirlItemRegex.FindAllStringSubmatch(xmlContent, -1)
	var results []TorrentSearchResult

	qTokens := strings.Fields(strings.ToLower(query))

	for _, it := range items {
		if len(it) < 2 {
			continue
		}
		itemContent := it[1]

		titleMatch := fitgirlTitleRegex.FindStringSubmatch(itemContent)
		if len(titleMatch) < 2 {
			continue
		}
		rawTitle := html.UnescapeString(strings.TrimSpace(titleMatch[1]))

		// Skip "Updates Digest" posts that don't represent a specific game repack
		if strings.HasPrefix(strings.ToLower(rawTitle), "updates digest") {
			continue
		}

		if !containsAllTokens(strings.ToLower(rawTitle), qTokens) {
			continue
		}

		magMatch := fitgirlMagnetRegex.FindString(itemContent)
		if magMatch == "" {
			continue
		}

		mag := html.UnescapeString(magMatch)
		ih := ParseInfoHashFromMagnet(mag)
		if ih == "" {
			continue
		}

		results = append(results, TorrentSearchResult{
			Title:     rawTitle,
			InfoHash:  ih,
			Magnet:    EnrichMagnetWithTrackers(mag),
			SizeBytes: 0,
			Seeders:   0,
			Leechers:  0,
			Source:    "FitGirl",
			Category:  "games",
		})
	}
	return results
}

func search1337x(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	mirrors := []string{
		"1337x.torrentbay.to",
		"1337x.to",
		"1337x.st",
	}

	q := strings.TrimSpace(query)
	var path string
	if q != "" {
		escaped := strings.ReplaceAll(url.QueryEscape(q), "+", "%20")
		switch cat {
		case "movies":
			path = fmt.Sprintf("/category-search/%s/Movies/%d/", escaped, page)
		case "tv":
			path = fmt.Sprintf("/category-search/%s/TV/%d/", escaped, page)
		case "games":
			path = fmt.Sprintf("/category-search/%s/Games/%d/", escaped, page)
		default:
			path = fmt.Sprintf("/search/%s/%d/", escaped, page)
		}
	} else {
		switch cat {
		case "movies":
			path = "/popular-movies"
		case "tv":
			path = "/popular-tv"
		case "games":
			path = "/popular-games"
		default:
			path = "/trending"
		}
	}

	for _, host := range mirrors {
		endpoint := "https://" + host + path
		reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		bodyBytes, err := fetchHTML(reqCtx, endpoint, uaAnpan)
		cancel()
		if err != nil {
			continue
		}

		results := parse1337xHTML(ctx, host, string(bodyBytes), cat)
		if len(results) > 0 {
			return results, nil
		}
	}

	return nil, nil
}

var leetxLinkRegex = regexp.MustCompile(`href="(/torrent/\d+/[^"]+)"[^>]*>([^<]+)</a>`)
var leetxSeedsRegex = regexp.MustCompile(`class="coll-2 seeds[^"]*">\s*(\d+)`)
var leetxLeechsRegex = regexp.MustCompile(`class="coll-3 leeches[^"]*">\s*(\d+)`)
var leetxSizeRegex = regexp.MustCompile(`class="coll-4 size[^"]*">\s*([\d.]+\s*[KMGT]i?B)`)
var leetxMagnetRegex = regexp.MustCompile(`href="(magnet:\?xt=urn:btih:[^"]+)"`)

type leetxRow struct {
	title     string
	path      string
	sizeBytes int64
	seeders   int
	leechers  int
}

func parse1337xHTML(ctx context.Context, host string, htmlContent string, cat string) []TorrentSearchResult {
	tableStart := strings.Index(htmlContent, "table-list")
	if tableStart < 0 {
		return nil
	}

	rawRows := strings.Split(htmlContent[tableStart:], "<tr")
	if len(rawRows) <= 1 {
		return nil
	}

	var rows []leetxRow
	for _, tr := range rawRows[1:] {
		linkM := leetxLinkRegex.FindStringSubmatch(tr)
		if len(linkM) < 3 {
			continue
		}
		path := linkM[1]
		title := html.UnescapeString(strings.TrimSpace(linkM[2]))

		seeds := 0
		if sM := leetxSeedsRegex.FindStringSubmatch(tr); len(sM) >= 2 {
			seeds, _ = strconv.Atoi(sM[1])
		}

		leechs := 0
		if lM := leetxLeechsRegex.FindStringSubmatch(tr); len(lM) >= 2 {
			leechs, _ = strconv.Atoi(lM[1])
		}

		var sizeBytes int64
		if szM := leetxSizeRegex.FindStringSubmatch(tr); len(szM) >= 2 {
			sizeBytes = units.ParseBytes(szM[1])
		}

		rows = append(rows, leetxRow{
			title:     title,
			path:      path,
			sizeBytes: sizeBytes,
			seeders:   seeds,
			leechers:  leechs,
		})
	}

	if len(rows) == 0 {
		return nil
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].seeders > rows[j].seeders
	})

	// Concurrently resolve magnets for top rows (up to 8)
	maxFetch := min(len(rows), 8)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []TorrentSearchResult

	for i := 0; i < maxFetch; i++ {
		row := rows[i]
		wg.Add(1)
		go func(r leetxRow) {
			defer wg.Done()
			detailURL := "https://" + host + r.path
			reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			detailBytes, err := fetchHTML(reqCtx, detailURL, uaAnpan)
			if err != nil {
				return
			}

			magM := leetxMagnetRegex.FindStringSubmatch(string(detailBytes))
			if len(magM) < 2 {
				return
			}
			rawMag := html.UnescapeString(magM[1])
			ih := ParseInfoHashFromMagnet(rawMag)
			if ih == "" {
				return
			}

			itemCat := cat
			if itemCat == "" || itemCat == "all" {
				lower := strings.ToLower(r.title)
				if strings.Contains(lower, "s0") || strings.Contains(lower, "season") || strings.Contains(lower, "episode") {
					itemCat = "tv"
				} else {
					itemCat = "movies"
				}
			}

			mu.Lock()
			results = append(results, TorrentSearchResult{
				Title:     r.title,
				InfoHash:  ih,
				Magnet:    EnrichMagnetWithTrackers(rawMag),
				SizeBytes: r.sizeBytes,
				Seeders:   r.seeders,
				Leechers:  r.leechers,
				Source:    "1337x",
				Category:  itemCat,
			})
			mu.Unlock()
		}(row)
	}

	wg.Wait()
	return results
}

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

func parseFlexibleInt(val interface{}) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

func parseFlexibleInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	case json.Number:
		n, _ := v.Int64()
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

	// Route based on category
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

	// Deduplicate by InfoHash
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

	// Sort
	switch strings.ToLower(opts.SortBy) {
	case "size", "size-desc":
		sort.Slice(deduped, func(i, j int) bool {
			return deduped[i].SizeBytes > deduped[j].SizeBytes
		})
	case "size-asc", "size-up", "smallest":
		sort.Slice(deduped, func(i, j int) bool {
			if deduped[i].SizeBytes <= 0 && deduped[j].SizeBytes > 0 {
				return false
			}
			if deduped[i].SizeBytes > 0 && deduped[j].SizeBytes <= 0 {
				return true
			}
			if deduped[i].SizeBytes == deduped[j].SizeBytes {
				return deduped[i].Seeders > deduped[j].Seeders
			}
			return deduped[i].SizeBytes < deduped[j].SizeBytes
		})
	case "peers", "leechers", "activity":
		sort.Slice(deduped, func(i, j int) bool {
			pi := deduped[i].Seeders + deduped[i].Leechers
			pj := deduped[j].Seeders + deduped[j].Leechers
			if pi == pj {
				return deduped[i].Seeders > deduped[j].Seeders
			}
			return pi > pj
		})
	case "source":
		sort.Slice(deduped, func(i, j int) bool {
			si := strings.ToLower(deduped[i].Source)
			sj := strings.ToLower(deduped[j].Source)
			if si == sj {
				return deduped[i].Seeders > deduped[j].Seeders
			}
			return si < sj
		})
	case "name":
		sort.Slice(deduped, func(i, j int) bool {
			return strings.ToLower(deduped[i].Title) < strings.ToLower(deduped[j].Title)
		})
	default: // "seeds"
		sort.Slice(deduped, func(i, j int) bool {
			if deduped[i].Seeders == deduped[j].Seeders {
				return deduped[i].SizeBytes > deduped[j].SizeBytes
			}
			return deduped[i].Seeders > deduped[j].Seeders
		})
	}

	if opts.Limit > 0 && len(deduped) > opts.Limit {
		deduped = deduped[:opts.Limit]
	}

	return deduped, nil
}

// 1. SubsPlease search
func searchSubsPlease(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	if page < 1 {
		page = 1
	}

	var endpoint string
	if strings.TrimSpace(query) == "" {
		endpoint = fmt.Sprintf("https://subsplease.org/api/?f=latest&tz=Asia/Tokyo&p=%d", page)
	} else {
		endpoint = fmt.Sprintf("https://subsplease.org/api/?f=search&tz=Asia/Tokyo&s=%s&p=%d", url.QueryEscape(query), page)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) anpan/1.0.0")

	resp, err := searchHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subsplease status: %d", resp.StatusCode)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "[]" || trimmed == "{}" || strings.Contains(trimmed, `"error"`) {
		return nil, nil
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
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
	if err == nil {
		if xl := u.Query().Get("xl"); xl != "" {
			if n, err := strconv.ParseInt(xl, 10, 64); err == nil && n > 0 {
				return n
			}
		}
	}
	idx := strings.Index(mag, "xl=")
	if idx != -1 {
		rest := mag[idx+3:]
		end := strings.IndexAny(rest, "&#")
		if end != -1 {
			rest = rest[:end]
		}
		if n, err := strconv.ParseInt(rest, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// 2. Nyaa RSS Search
type nyaaRSS struct {
	Channel struct {
		Items []struct {
			Title      string `xml:"title"`
			Link       string `xml:"link"`
			Guid       string `xml:"guid"`
			Seeders    int    `xml:"seeders"`
			Leechers   int    `xml:"leechers"`
			SizeStr    string `xml:"size"`
			InfoHash   string `xml:"infoHash"`
			CategoryId string `xml:"categoryId"`
		} `xml:"item"`
	} `xml:"channel"`
}

func searchNyaa(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	if page < 1 {
		page = 1
	}

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
		req, err := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) anpan/1.0.0")

		resp, err := searchHTTPClient.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			lastErr = fmt.Errorf("nyaa status: %d", resp.StatusCode)
			continue
		}

		var feed nyaaRSS
		err = xml.NewDecoder(resp.Body).Decode(&feed)
		resp.Body.Close()
		cancel()

		if err != nil {
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

// 3. YTS Movies Search
type ytsResponse struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Data          struct {
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
	if page < 1 {
		page = 1
	}
	mirrors := []string{
		"https://movies-api.accel.li/api/v2/list_movies.json?limit=50&page=%d&query_term=%s",
		"https://yts.mx/api/v2/list_movies.json?limit=50&page=%d&query_term=%s",
	}

	var lastErr error
	for _, mirror := range mirrors {
		endpoint := fmt.Sprintf(mirror, page, url.QueryEscape(query))
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "anpan/torlink-client")

		resp, err := searchHTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		var ytsData ytsResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&ytsData)
		resp.Body.Close()

		if decodeErr != nil || ytsData.Data.MovieCount == 0 {
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

// 4. EZTV TV Shows Search
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
	TorrentsCount int               `json:"torrents_count"`
	Torrents      []eztvTorrentItem `json:"torrents"`
}

func searchEZTV(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	if page < 1 {
		page = 1
	}
	endpoint := fmt.Sprintf("https://eztvx.to/api/get-torrents?limit=100&page=%d", page)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "anpan/torlink-client")

	resp, err := searchHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eztv status: %d", resp.StatusCode)
	}

	var data eztvResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	qTokens := strings.Fields(strings.ToLower(query))

	var results []TorrentSearchResult
	for _, t := range data.Torrents {
		title := t.Title
		if title == "" {
			title = t.Filename
		}
		if len(qTokens) > 0 {
			lower := strings.ToLower(title + " " + t.Filename)
			matched := true
			for _, tok := range qTokens {
				if !strings.Contains(lower, tok) {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
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

// 5. The Pirate Bay / apibay Search
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
	if page < 1 {
		page = 1
	}

	var endpoint string
	if strings.TrimSpace(query) == "" {
		switch cat {
		case "anime":
			endpoint = "https://apibay.org/precompiled/data_top100_205.json"
		case "tv":
			endpoint = "https://apibay.org/precompiled/data_top100_208.json"
		case "movies":
			endpoint = "https://apibay.org/precompiled/data_top100_207.json"
		case "games":
			endpoint = "https://apibay.org/precompiled/data_top100_400.json"
		default:
			endpoint = "https://apibay.org/precompiled/data_top100_all.json"
		}
	} else {
		catParam := ""
		switch cat {
		case "anime":
			catParam = "&cat=205"
		case "tv":
			catParam = "&cat=208"
		case "movies":
			catParam = "&cat=207"
		case "games":
			catParam = "&cat=400"
		}
		endpoint = fmt.Sprintf("https://apibay.org/q.php?q=%s%s", url.QueryEscape(query), catParam)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "anpan/torlink-client")

	resp, err := searchHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apibay status: %d", resp.StatusCode)
	}

	var items []apibayItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
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

// 6. FitGirl Repacks Search
func searchFitGirl(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	if page < 1 {
		page = 1
	}

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

	req, err := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
	if err != nil {
		return nil, nil
	}
	req.Header.Set("User-Agent", "torlink (+https://www.npmjs.com/package/torlnk)")

	resp, err := searchHTTPClient.Do(req)
	if err != nil {
		return nil, nil // graceful fail-soft on DDoS-Guard / network timeouts
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
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

		if len(qTokens) > 0 {
			lower := strings.ToLower(rawTitle)
			matched := true
			for _, tok := range qTokens {
				if !strings.Contains(lower, tok) {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
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

// 7. 1337x Search
func search1337x(ctx context.Context, query string, cat string, page int) ([]TorrentSearchResult, error) {
	if page < 1 {
		page = 1
	}

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
		req, err := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
		if err != nil {
			cancel()
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) anpan/1.0.0")

		resp, err := searchHTTPClient.Do(req)
		if err != nil {
			cancel()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
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

	// Sort rows by seeders descending
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].seeders > rows[j].seeders
	})

	// Concurrently resolve magnets for top rows (up to 8)
	maxFetch := len(rows)
	if maxFetch > 8 {
		maxFetch = 8
	}

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

			req, err := http.NewRequestWithContext(reqCtx, "GET", detailURL, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) anpan/1.0.0")

			resp, err := searchHTTPClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return
			}

			detailBytes, err := io.ReadAll(resp.Body)
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

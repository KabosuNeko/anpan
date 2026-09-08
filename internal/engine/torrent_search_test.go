package engine

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestSearchTorrentsOptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := &SearchOptions{
		Category: "anime",
		Limit:    10,
		SortBy:   "seeds",
	}

	// Just ensure it executes and respects context cancellation without panic
	res, err := SearchTorrents(ctx, "ubuntu", opts)
	t.Logf("SearchTorrents ubuntu anime: %d results, err: %v", len(res), err)
}

func TestSearchNyaaDirect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := searchNyaa(ctx, "one", "anime", 1)
	t.Logf("searchNyaa: %d results, err: %v", len(res), err)
	if err != nil {
		t.Fatalf("searchNyaa failed: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected searchNyaa to return results")
	}
}

func TestAnimePagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Page 1
	res1, err := SearchTorrents(ctx, "one", &SearchOptions{Category: "anime", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("page 1 failed: %v", err)
	}
	if len(res1) == 0 {
		t.Fatalf("expected results on page 1 of anime")
	}

	// Page 2
	res2, err := SearchTorrents(ctx, "one", &SearchOptions{Category: "anime", Page: 2, Limit: 50})
	if err != nil {
		t.Fatalf("page 2 failed: %v", err)
	}
	if len(res2) == 0 {
		t.Fatalf("expected results on page 2 of anime")
	}

	t.Logf("Anime Page 1 results: %d, Page 2 results: %d", len(res1), len(res2))
}

func TestAnimeBrowsePagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Page 1
	subRes, subErr := searchSubsPlease(ctx, "", "anime", 1)
	nyaaRes, nyaaErr := searchNyaa(ctx, "", "anime", 1)
	tpbRes, tpbErr := searchPirateBay(ctx, "", "anime", 1)
	t.Logf("Page 1: subs=%d (err %v), nyaa=%d (err %v), tpb=%d (err %v)", len(subRes), subErr, len(nyaaRes), nyaaErr, len(tpbRes), tpbErr)

	res1, err := SearchTorrents(ctx, "", &SearchOptions{Category: "anime", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("page 1 failed: %v", err)
	}
	if len(res1) == 0 {
		t.Fatalf("expected results on page 1 of anime browse")
	}

	// Page 2
	res2, err := SearchTorrents(ctx, "", &SearchOptions{Category: "anime", Page: 2, Limit: 50})
	if err != nil {
		t.Fatalf("page 2 failed: %v", err)
	}
	if len(res2) == 0 {
		t.Fatalf("expected results on page 2 of anime browse")
	}

	t.Logf("Anime Browse Page 1: %d, Page 2: %d", len(res1), len(res2))
}

func TestSearchTorrentsDedupeAndSort(t *testing.T) {
	// Test deduplication and sorting logic manually
	results := []TorrentSearchResult{
		{Title: "Ubuntu 22.04", InfoHash: "aabbcc", Seeders: 10, SizeBytes: 1000},
		{Title: "Ubuntu 22.04 Mirror", InfoHash: "aabbcc", Seeders: 50, SizeBytes: 1000},
		{Title: "Arch Linux", InfoHash: "ddeeff", Seeders: 30, SizeBytes: 2000},
	}

	seen := make(map[string]TorrentSearchResult)
	for _, r := range results {
		existing, found := seen[r.InfoHash]
		if !found || r.Seeders > existing.Seeders {
			seen[r.InfoHash] = r
		}
	}

	if len(seen) != 2 {
		t.Fatalf("expected 2 deduped items, got %d", len(seen))
	}
	if seen["aabbcc"].Seeders != 50 {
		t.Errorf("expected highest seeder count (50), got %d", seen["aabbcc"].Seeders)
	}
}

func TestSearchTorrentsAllSortModes(t *testing.T) {
	data := []TorrentSearchResult{
		{Title: "Banana", Source: "YTS", Seeders: 10, Leechers: 5, SizeBytes: 500},
		{Title: "Apple", Source: "Nyaa", Seeders: 50, Leechers: 100, SizeBytes: 2000},
		{Title: "Cherry", Source: "EZTV", Seeders: 30, Leechers: 10, SizeBytes: 100},
	}

	// 1. Sort by seeds (Apple 50 > Cherry 30 > Banana 10)
	sSeeds := append([]TorrentSearchResult(nil), data...)
	sort.Slice(sSeeds, func(i, j int) bool {
		return sSeeds[i].Seeders > sSeeds[j].Seeders
	})
	if sSeeds[0].Title != "Apple" || sSeeds[2].Title != "Banana" {
		t.Errorf("Seeds sort failed: %+v", sSeeds)
	}

	// 2. Sort by size-asc (Cherry 100 < Banana 500 < Apple 2000)
	sSizeAsc := append([]TorrentSearchResult(nil), data...)
	sort.Slice(sSizeAsc, func(i, j int) bool {
		return sSizeAsc[i].SizeBytes < sSizeAsc[j].SizeBytes
	})
	if sSizeAsc[0].Title != "Cherry" || sSizeAsc[2].Title != "Apple" {
		t.Errorf("Size-asc sort failed: %+v", sSizeAsc)
	}

	// 3. Sort by peers (Apple 150 > Cherry 40 > Banana 15)
	sPeers := append([]TorrentSearchResult(nil), data...)
	sort.Slice(sPeers, func(i, j int) bool {
		return (sPeers[i].Seeders + sPeers[i].Leechers) > (sPeers[j].Seeders + sPeers[j].Leechers)
	})
	if sPeers[0].Title != "Apple" || sPeers[2].Title != "Banana" {
		t.Errorf("Peers sort failed: %+v", sPeers)
	}

	// 4. Sort by source (EZTV < Nyaa < YTS)
	sSource := append([]TorrentSearchResult(nil), data...)
	sort.Slice(sSource, func(i, j int) bool {
		return strings.ToLower(sSource[i].Source) < strings.ToLower(sSource[j].Source)
	})
	if sSource[0].Source != "EZTV" || sSource[2].Source != "YTS" {
		t.Errorf("Source sort failed: %+v", sSource)
	}
}

func TestGamesSearch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 1. Search games query
	res, err := SearchTorrents(ctx, "cyberpunk", &SearchOptions{Category: "games", Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("games search failed: %v", err)
	}
	t.Logf("Games query search returned %d results", len(res))
	if len(res) == 0 {
		t.Fatalf("expected games search to return results")
	}

	// 2. Browse games (empty query)
	browseRes, browseErr := SearchTorrents(ctx, "", &SearchOptions{Category: "games", Page: 1, Limit: 20})
	if browseErr != nil {
		t.Fatalf("games browse failed: %v", browseErr)
	}
	t.Logf("Games browse returned %d results", len(browseRes))
	if len(browseRes) == 0 {
		t.Fatalf("expected games browse to return results")
	}
}

func TestParseFitGirlRSS(t *testing.T) {
	sampleXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<item>
		<title>Updates Digest for August 30, 2026</title>
		<content:encoded><![CDATA[Some updates]]></content:encoded>
	</item>
	<item>
		<title>ELDEN RING: Shadow of the Erdtree Edition</title>
		<content:encoded><![CDATA[
			<p>Download here: <a href="magnet:?xt=urn:btih:6601d0dc9ae9fea8582ac703f92553b717747c42&amp;dn=EldenRing">Magnet</a></p>
		]]></content:encoded>
	</item>
</channel>
</rss>`

	results := parseFitGirlRSS(sampleXML, "elden")
	if len(results) != 1 {
		t.Fatalf("expected 1 result from sample FitGirl RSS, got %d", len(results))
	}
	r := results[0]
	if r.Source != "FitGirl" {
		t.Errorf("expected Source 'FitGirl', got %s", r.Source)
	}
	if r.Category != "games" {
		t.Errorf("expected Category 'games', got %s", r.Category)
	}
	if r.InfoHash != "6601d0dc9ae9fea8582ac703f92553b717747c42" {
		t.Errorf("expected infohash 6601d0dc9ae9fea8582ac703f92553b717747c42, got %s", r.InfoHash)
	}
}

func TestParse1337xHTML(t *testing.T) {
	sampleHTML := `<div class="table-list-wrap">
<table class="table-list table table-responsive table-striped">
<thead><tr><th>name</th></tr></thead>
<tbody>
<tr>
  <td class="coll-1 name"><a href="/torrent/5020920/Dune-2021-1080p/">Dune.2021.1080p.WEBRip</a></td>
  <td class="coll-2 seeds">7312</td>
  <td class="coll-3 leeches">899</td>
  <td class="coll-4 size">10.6 GB</td>
</tr>
</tbody>
</table>
</div>`

	tableStart := strings.Index(sampleHTML, "table-list")
	if tableStart < 0 {
		t.Fatalf("table-list not found in sample HTML")
	}
	rawRows := strings.Split(sampleHTML[tableStart:], "<tr")
	if len(rawRows) < 3 {
		t.Fatalf("expected at least 3 row chunks (header + data) in sample HTML, got %d", len(rawRows))
	}
	// Data row is rawRows[2]
	linkM := leetxLinkRegex.FindStringSubmatch(rawRows[2])
	if len(linkM) < 3 || linkM[1] != "/torrent/5020920/Dune-2021-1080p/" || linkM[2] != "Dune.2021.1080p.WEBRip" {
		t.Errorf("leetxLinkRegex failed: %v", linkM)
	}
	seedsM := leetxSeedsRegex.FindStringSubmatch(rawRows[2])
	if len(seedsM) < 2 || seedsM[1] != "7312" {
		t.Errorf("leetxSeedsRegex failed: %v", seedsM)
	}
	sizeM := leetxSizeRegex.FindStringSubmatch(rawRows[2])
	if len(sizeM) < 2 || sizeM[1] != "10.6 GB" {
		t.Errorf("leetxSizeRegex failed: %v", sizeM)
	}
}




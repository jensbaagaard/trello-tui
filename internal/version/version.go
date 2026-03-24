package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var repoURL = "https://api.github.com/repos/jensbaagaard/trello-tui/tags?per_page=10"

type ghTag struct {
	Name string `json:"name"`
}

func CheckLatest() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(repoURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var tags []ghTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return ""
	}

	var best string
	var bestParts []int
	for _, t := range tags {
		parts := parseSemver(t.Name)
		if parts == nil {
			continue
		}
		if bestParts == nil || compareSemver(parts, bestParts) > 0 {
			best = t.Name
			bestParts = parts
		}
	}
	return best
}

func IsNewer(latest, current string) bool {
	l := parseSemver(latest)
	c := parseSemver(current)
	if l == nil || c == nil {
		return false
	}
	return compareSemver(l, c) > 0
}

func FormatNotice(latest string) string {
	return fmt.Sprintf("Update available: %s — https://github.com/jensbaagaard/trello-tui/releases", latest)
}

func parseSemver(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}

func compareSemver(a, b []int) int {
	for i := 0; i < 3; i++ {
		if a[i] > b[i] {
			return 1
		}
		if a[i] < b[i] {
			return -1
		}
	}
	return 0
}

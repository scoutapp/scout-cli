package cmd

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var (
	showAllApps bool
	byDay       bool
	byApp       bool
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show transaction usage across all apps",
	Run:   runUsage,
}

func init() {
	usageCmd.Flags().BoolVar(&showAllApps, "all", false, "Include apps with zero usage")
	usageCmd.Flags().BoolVar(&byDay, "by-day", false, "Show daily transaction breakdown")
	usageCmd.Flags().BoolVar(&byApp, "by-app", false, "Break down by app (use with --by-day)")
	rootCmd.AddCommand(usageCmd)
}

type appUsage struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Transactions float64 `json:"transactions"`
}

type dailyUsage struct {
	Date         string `json:"date"`
	Transactions int64  `json:"transactions"`
	TopEndpoint  string `json:"top_endpoint,omitempty"`
}

type dailyAppUsage struct {
	AppID        int    `json:"app_id"`
	AppName      string `json:"app_name"`
	Transactions int64  `json:"transactions"`
	TopEndpoint  string `json:"top_endpoint,omitempty"`
}

type dailyReport struct {
	Date  string          `json:"date"`
	Total int64           `json:"total"`
	Apps  []dailyAppUsage `json:"apps"`
}

func runUsage(cmd *cobra.Command, args []string) {
	if byDay {
		runUsageByDay(cmd, args)
		return
	}

	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	from, to, err := resolveTimeframe()
	if err != nil {
		exitError(err.Error())
	}

	apps, err := client.ListApps()
	if err != nil {
		handleAPIError(err)
		return
	}

	chunks := splitTimeframe(from, to)

	results := fetchAllApps(apps, func(a api.App) appUsage {
		return appUsage{
			ID:           a.ID,
			Name:         a.Name,
			Transactions: fetchAppTransactions(client, a.ID, chunks),
		}
	})

	if !showAllApps {
		filtered := results[:0]
		for _, r := range results {
			if r.Transactions > 0 {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Transactions > results[j].Transactions
	})

	if jsonOutput {
		outputJSON(results)
		return
	}

	printTimeframe(from, to)

	var grandTotal float64
	for _, r := range results {
		grandTotal += r.Transactions
	}

	total := len(results)
	limit, _ := applyLimit(total)

	headers := []string{"Name", "Transactions", "% of Total"}
	rows := make([][]string, limit)
	for i := 0; i < limit; i++ {
		r := results[i]
		pct := 0.0
		if grandTotal > 0 {
			pct = (r.Transactions / grandTotal) * 100
		}
		rows[i] = []string{
			r.Name,
			formatTransactions(r.Transactions),
			fmt.Sprintf("%.1f%%", pct),
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(limit, total)
	printTotalFooter(grandTotal)
}

func runUsageByDay(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	from, to, err := resolveTimeframe()
	if err != nil {
		exitError(err.Error())
	}

	if appID > 0 {
		runUsageByDaySingleApp(client, appID, from, to)
	} else if byApp {
		runUsageByDayByApp(client, from, to)
	} else {
		runUsageByDayAllApps(client, from, to)
	}
}

func runUsageByDayAllApps(client *api.Client, from, to string) {
	apps, err := client.ListApps()
	if err != nil {
		handleAPIError(err)
		return
	}

	chunks := splitTimeframe(from, to)
	allPoints := fetchAllAppPoints(client, apps, chunks)
	days := bucketByDay(allPoints)

	if jsonOutput {
		outputJSON(days)
		return
	}

	printTimeframe(from, to)

	total := len(days)
	limit, _ := applyLimit(total)

	headers := []string{"Day", "Transactions"}
	rows := make([][]string, limit)
	var grandTotal int64
	for i := 0; i < limit; i++ {
		d := days[i]
		grandTotal += d.Transactions
		rows[i] = []string{
			d.Date,
			formatTransactions(float64(d.Transactions)),
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(limit, total)
	printTotalFooter(float64(grandTotal))
}

func runUsageByDayByApp(client *api.Client, from, to string) {
	apps, err := client.ListApps()
	if err != nil {
		handleAPIError(err)
		return
	}

	chunks := splitTimeframe(from, to)

	type appPointsResult struct {
		app    api.App
		points []api.MetricPoint
	}

	appData := fetchAllApps(apps, func(a api.App) appPointsResult {
		return appPointsResult{app: a, points: fetchAppPoints(client, a.ID, chunks)}
	})

	// Bucket each app's points by day using the shared interval logic
	type appDayInfo struct {
		appID   int
		appName string
		txns    float64
	}
	dayMap := make(map[string][]appDayInfo)

	for _, ad := range appData {
		dayTotals := sumIntervalsByDay(ad.points)
		for day, txns := range dayTotals {
			found := false
			for j := range dayMap[day] {
				if dayMap[day][j].appID == ad.app.ID {
					dayMap[day][j].txns += txns
					found = true
					break
				}
			}
			if !found {
				dayMap[day] = append(dayMap[day], appDayInfo{
					appID:   ad.app.ID,
					appName: ad.app.Name,
					txns:    txns,
				})
			}
		}
	}

	dates := sortedKeys(dayMap)

	// Fetch top endpoints per app per day in parallel
	type endpointKey struct {
		date  string
		appID int
	}
	topEndpoints := make(map[endpointKey]string)
	var epMu sync.Mutex
	var epWg sync.WaitGroup

	for _, date := range dates {
		for _, info := range dayMap[date] {
			if info.txns <= 0 {
				continue
			}
			epWg.Add(1)
			go func(d string, aID int) {
				defer epWg.Done()
				dayStart := d + "T00:00:00Z"
				dayEnd := d + "T23:59:59Z"
				endpoints, err := client.ListEndpoints(aID, dayStart, dayEnd)
				if err != nil || len(endpoints) == 0 {
					return
				}
				epMu.Lock()
				topEndpoints[endpointKey{date: d, appID: aID}] = endpoints[0].Name
				epMu.Unlock()
			}(date, info.appID)
		}
	}
	epWg.Wait()

	reports := make([]dailyReport, 0, len(dates))
	for _, date := range dates {
		appInfos := dayMap[date]
		sort.Slice(appInfos, func(i, j int) bool {
			return appInfos[i].txns > appInfos[j].txns
		})

		var dayTotal int64
		appUsages := make([]dailyAppUsage, 0, len(appInfos))
		for _, info := range appInfos {
			txns := int64(math.Round(info.txns))
			if txns == 0 && !showAllApps {
				continue
			}
			dayTotal += txns
			appUsages = append(appUsages, dailyAppUsage{
				AppID:        info.appID,
				AppName:      info.appName,
				Transactions: txns,
				TopEndpoint:  topEndpoints[endpointKey{date: date, appID: info.appID}],
			})
		}
		reports = append(reports, dailyReport{
			Date:  date,
			Total: dayTotal,
			Apps:  appUsages,
		})
	}

	if jsonOutput {
		outputJSON(reports)
		return
	}

	printTimeframe(from, to)

	var grandTotal int64
	for _, report := range reports {
		grandTotal += report.Total
	}

	for _, report := range reports {
		fmt.Printf("%s %s\n", output.HeaderStyle.Render("── "+report.Date+" ──"),
			output.DimStyle.Render(fmt.Sprintf("%s transactions", formatTransactions(float64(report.Total)))))

		headers := []string{"App", "Transactions", "% of Day", "% of Total", "Top Endpoint"}
		rows := make([][]string, 0, len(report.Apps))
		for _, a := range report.Apps {
			pctDay := 0.0
			if report.Total > 0 {
				pctDay = (float64(a.Transactions) / float64(report.Total)) * 100
			}
			pctTotal := 0.0
			if grandTotal > 0 {
				pctTotal = (float64(a.Transactions) / float64(grandTotal)) * 100
			}
			rows = append(rows, []string{
				a.AppName,
				formatTransactions(float64(a.Transactions)),
				fmt.Sprintf("%.1f%%", pctDay),
				fmt.Sprintf("%.1f%%", pctTotal),
				a.TopEndpoint,
			})
		}
		fmt.Println(output.RenderTable(headers, rows))
	}

	if grandTotal > 0 {
		fmt.Printf("Total: %s transactions\n", formatTransactions(float64(grandTotal)))
	}
}

func runUsageByDaySingleApp(client *api.Client, id int, from, to string) {
	chunks := splitTimeframe(from, to)
	points := fetchAppPoints(client, id, chunks)
	days := bucketByDay(points)

	// Fetch top endpoint for each day in parallel
	var wg sync.WaitGroup
	for i := range days {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			d := days[idx]
			dayStart := d.Date + "T00:00:00Z"
			dayEnd := d.Date + "T23:59:59Z"
			endpoints, err := client.ListEndpoints(id, dayStart, dayEnd)
			if err != nil || len(endpoints) == 0 {
				return
			}
			days[idx].TopEndpoint = endpoints[0].Name
		}(i)
	}
	wg.Wait()

	if jsonOutput {
		outputJSON(days)
		return
	}

	printTimeframe(from, to)

	total := len(days)
	limit, _ := applyLimit(total)

	headers := []string{"Day", "Transactions", "Top Endpoint"}
	rows := make([][]string, limit)
	var grandTotal int64
	for i := 0; i < limit; i++ {
		d := days[i]
		grandTotal += d.Transactions
		rows[i] = []string{
			d.Date,
			formatTransactions(float64(d.Transactions)),
			d.TopEndpoint,
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(limit, total)
	printTotalFooter(float64(grandTotal))
}

// fetchAllApps runs fn for each app in parallel and collects the results.
func fetchAllApps[T any](apps []api.App, fn func(api.App) T) []T {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []T
	)
	for _, app := range apps {
		wg.Add(1)
		go func(a api.App) {
			defer wg.Done()
			result := fn(a)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(app)
	}
	wg.Wait()
	return results
}

// fetchAllAppPoints fetches and merges throughput points for all apps in parallel.
func fetchAllAppPoints(client *api.Client, apps []api.App, chunks [][2]string) []api.MetricPoint {
	perApp := fetchAllApps(apps, func(a api.App) []api.MetricPoint {
		return fetchAppPoints(client, a.ID, chunks)
	})
	var allPoints []api.MetricPoint
	for _, pts := range perApp {
		allPoints = append(allPoints, pts...)
	}
	return allPoints
}

// fetchAppPoints fetches throughput time-series data across all time chunks.
func fetchAppPoints(client *api.Client, appID int, chunks [][2]string) []api.MetricPoint {
	var allPoints []api.MetricPoint
	for _, chunk := range chunks {
		metrics, err := client.GetMetrics(appID, "throughput", chunk[0], chunk[1])
		if err != nil {
			continue
		}
		allPoints = append(allPoints, metrics.Series["throughput"]...)
	}
	return allPoints
}

// intervalTransactions computes the transaction count for a consecutive pair
// of metric points (RPM * interval in minutes). Returns 0 if the timestamps
// are invalid or the interval is non-positive.
func intervalTransactions(prev, curr api.MetricPoint) float64 {
	t0, err0 := time.Parse(time.RFC3339, prev.Timestamp)
	t1, err1 := time.Parse(time.RFC3339, curr.Timestamp)
	if err0 != nil || err1 != nil {
		return 0
	}
	intervalMinutes := t1.Sub(t0).Minutes()
	if intervalMinutes <= 0 {
		return 0
	}
	return prev.Value * intervalMinutes
}

// sumIntervalsByDay computes per-day transaction totals from RPM time-series
// data. Returns a map of date strings to transaction counts.
func sumIntervalsByDay(points []api.MetricPoint) map[string]float64 {
	dayTotals := make(map[string]float64)
	for i := 1; i < len(points); i++ {
		txns := intervalTransactions(points[i-1], points[i])
		if txns <= 0 {
			continue
		}
		t0, _ := time.Parse(time.RFC3339, points[i-1].Timestamp)
		dayTotals[t0.Format("2006-01-02")] += txns
	}
	return dayTotals
}

// bucketByDay aggregates time-series RPM data into daily transaction counts.
func bucketByDay(points []api.MetricPoint) []dailyUsage {
	if len(points) < 2 {
		return nil
	}

	dayTotals := sumIntervalsByDay(points)

	days := make([]dailyUsage, 0, len(dayTotals))
	for day, txns := range dayTotals {
		days = append(days, dailyUsage{
			Date:         day,
			Transactions: int64(math.Round(txns)),
		})
	}

	sort.Slice(days, func(i, j int) bool {
		return days[i].Date < days[j].Date
	})

	return days
}

// splitTimeframe splits a time range into chunks of at most 14 days.
func splitTimeframe(from, to string) [][2]string {
	fromTime, _ := time.Parse(time.RFC3339, from)
	toTime, _ := time.Parse(time.RFC3339, to)

	maxChunk := 14 * 24 * time.Hour
	var chunks [][2]string

	for fromTime.Before(toTime) {
		chunkEnd := fromTime.Add(maxChunk)
		if chunkEnd.After(toTime) {
			chunkEnd = toTime
		}
		chunks = append(chunks, [2]string{
			fromTime.Format(time.RFC3339),
			chunkEnd.Format(time.RFC3339),
		})
		fromTime = chunkEnd
	}

	return chunks
}

// fetchAppTransactions fetches throughput data across all time chunks and
// computes total transaction count.
func fetchAppTransactions(client *api.Client, appID int, chunks [][2]string) float64 {
	return calculateTransactions(fetchAppPoints(client, appID, chunks))
}

// calculateTransactions computes total transactions from RPM time-series data.
// For each consecutive pair of points, it multiplies the RPM value by the
// interval in minutes.
func calculateTransactions(points []api.MetricPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	var total float64
	for i := 1; i < len(points); i++ {
		total += intervalTransactions(points[i-1], points[i])
	}
	return total
}

// printTimeframe prints the timeframe header for usage output.
func printTimeframe(from, to string) {
	fromTime, _ := time.Parse(time.RFC3339, from)
	toTime, _ := time.Parse(time.RFC3339, to)
	fromStr := fromTime.Format("Jan 02, 2006 15:04 UTC")
	toStr := toTime.Format("Jan 02, 2006 15:04 UTC")
	fmt.Printf("%s\n\n", output.DimStyle.Render(fmt.Sprintf("Timeframe: %s → %s", fromStr, toStr)))
}

// printTotalFooter prints the grand total line if non-zero.
func printTotalFooter(grandTotal float64) {
	if grandTotal > 0 {
		fmt.Printf("\nTotal: %s transactions\n", formatTransactions(grandTotal))
	}
}

// sortedKeys returns the keys of a map sorted in ascending order.
func sortedKeys[V any](m map[string][]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// formatTransactions formats a number with comma separators.
func formatTransactions(n float64) string {
	rounded := int64(math.Round(n))
	if rounded == 0 {
		return "0"
	}

	neg := rounded < 0
	if neg {
		rounded = -rounded
	}

	s := fmt.Sprintf("%d", rounded)
	parts := make([]byte, 0, len(s)+(len(s)-1)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			parts = append(parts, ',')
		}
		parts = append(parts, byte(c))
	}

	if neg {
		return "-" + string(parts)
	}
	return string(parts)
}

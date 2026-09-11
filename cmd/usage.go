package cmd

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var (
	showAllApps   bool
	byDay         bool
	byApp         bool
	billingPeriod bool
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show web transaction usage across all apps",
	Long: `Show web transaction usage across all apps for a timeframe.

Per-app figures are calculated from the throughput metric, which counts web
requests only; background job transactions are excluded. The billed total,
which includes background jobs, is shown by 'scout billing' and, alongside
the per-app breakdown, by 'scout usage --billing-period'.`,
	Run: runUsage,
}

func init() {
	usageCmd.Flags().BoolVar(&showAllApps, "all", false, "Include apps with zero usage")
	usageCmd.Flags().BoolVar(&byDay, "by-day", false, "Show daily transaction breakdown")
	usageCmd.Flags().BoolVar(&byApp, "by-app", false, "Break down by app (use with --by-day)")
	usageCmd.Flags().BoolVar(&billingPeriod, "billing-period", false, "Use current billing period as timeframe")
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

type appDayInfo struct {
	appID   int
	appName string
	txns    float64
}

type endpointKey struct {
	date  string
	appID int
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

	filterID, err := appIDFilter()
	if err != nil {
		exitError(err.Error())
	}

	tf, err := resolveUsageTimeframe(client)
	if err != nil {
		exitError(err.Error())
	}
	from, to := tf.from, tf.to

	apps, err := client.ListApps()
	if err != nil {
		handleAPIError(err)
		return
	}
	apps, err = filterApps(apps, filterID)
	if err != nil {
		exitError(err.Error())
	}

	chunks := splitTimeframe(from, to)

	results := output.RunWithProgress("Fetching usage data", !jsonOutput, func(update func(float64)) []appUsage {
		return fetchAllApps(apps, func(a api.App) appUsage {
			return appUsage{
				ID:           a.ID,
				Name:         a.Name,
				Transactions: fetchAppTransactions(client, a.ID, chunks),
			}
		}, func(done, total int) {
			update(float64(done) / float64(total))
		})
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

	// The grand total covers every app in the timeframe; -n only selects how
	// many rows are shown.
	var grandTotal float64
	for _, r := range results {
		grandTotal += r.Transactions
	}

	results, total := limitSlice(results)

	if usageStructuredOutput(tf, results) {
		return
	}

	printUsageHeader(tf)

	headers := []string{"Name", "Web Transactions", "% of Web"}
	rows := make([][]string, len(results))
	for i := range results {
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
	printTruncated(len(results), total)
	printTotalFooter(grandTotal)
	printServerTotal(tf)
}

// filterApps narrows the app list to a single app when --app was given.
func filterApps(apps []api.App, filterID int) ([]api.App, error) {
	if filterID <= 0 {
		return apps, nil
	}
	for _, a := range apps {
		if a.ID == filterID {
			return []api.App{a}, nil
		}
	}
	return nil, fmt.Errorf("app %d not found in this account", filterID)
}

func runUsageByDay(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	filterID, err := appIDFilter()
	if err != nil {
		exitError(err.Error())
	}

	tf, err := resolveUsageTimeframe(client)
	if err != nil {
		exitError(err.Error())
	}

	if filterID > 0 {
		runUsageByDaySingleApp(client, filterID, tf)
	} else if byApp {
		runUsageByDayByApp(client, tf)
	} else {
		runUsageByDayAllApps(client, tf)
	}
}

func runUsageByDayAllApps(client *api.Client, tf usageTimeframe) {
	from, to := tf.from, tf.to
	apps, err := client.ListApps()
	if err != nil {
		handleAPIError(err)
		return
	}

	chunks := splitTimeframe(from, to)

	allPoints := output.RunWithProgress("Fetching daily usage data", !jsonOutput, func(update func(float64)) []api.MetricPoint {
		return fetchAllAppPoints(client, apps, chunks, func(done, total int) {
			update(float64(done) / float64(total))
		})
	})
	days := bucketByDay(allPoints)

	days, total := limitSlice(days)

	if usageStructuredOutput(tf, days) {
		return
	}

	printUsageHeader(tf)

	headers := []string{"Day", "Web Transactions"}
	rows := make([][]string, len(days))
	var grandTotal int64
	for i := range days {
		d := days[i]
		grandTotal += d.Transactions
		rows[i] = []string{
			d.Date,
			formatTransactions(float64(d.Transactions)),
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(len(days), total)
	printTotalFooter(float64(grandTotal))
	printServerTotal(tf)
}

func runUsageByDayByApp(client *api.Client, tf usageTimeframe) {
	from, to := tf.from, tf.to
	apps, err := client.ListApps()
	if err != nil {
		handleAPIError(err)
		return
	}

	chunks := splitTimeframe(from, to)
	showProgress := !jsonOutput

	type appPointsResult struct {
		app    api.App
		points []api.MetricPoint
	}

	type byDayByAppResult struct {
		appData      []appPointsResult
		dayMap       map[string][]appDayInfo
		dates        []string
		topEndpoints map[endpointKey]string
	}

	result := output.RunWithProgress("Fetching usage data", showProgress, func(update func(float64)) byDayByAppResult {
		// Phase 1: Fetch app data (0% - 50%)
		appData := fetchAllApps(apps, func(a api.App) appPointsResult {
			return appPointsResult{app: a, points: fetchAppPoints(client, a.ID, chunks)}
		}, func(done, total int) {
			update(float64(done) / float64(total) * 0.5)
		})

		// Bucket each app's points by day using the shared interval logic
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

		// Phase 2: Fetch top endpoints (50% - 100%)
		topEndpoints := make(map[endpointKey]string)
		var epMu sync.Mutex
		var epWg sync.WaitGroup
		var epTotal int64
		var epDone int64

		for _, date := range dates {
			for _, info := range dayMap[date] {
				if info.txns > 0 {
					epTotal++
				}
			}
		}

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
						n := atomic.AddInt64(&epDone, 1)
						if epTotal > 0 {
							update(0.5 + float64(n)/float64(epTotal)*0.5)
						}
						return
					}
					epMu.Lock()
					topEndpoints[endpointKey{date: d, appID: aID}] = endpoints[0].Name
					epMu.Unlock()
					n := atomic.AddInt64(&epDone, 1)
					if epTotal > 0 {
						update(0.5 + float64(n)/float64(epTotal)*0.5)
					}
				}(date, info.appID)
			}
		}
		epWg.Wait()

		return byDayByAppResult{
			appData:      appData,
			dayMap:       dayMap,
			dates:        dates,
			topEndpoints: topEndpoints,
		}
	})

	dayMap := result.dayMap
	dates := result.dates
	topEndpoints := result.topEndpoints

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

	// The grand total covers every day in the timeframe; -n only selects how
	// many days are shown.
	var grandTotal int64
	for _, report := range reports {
		grandTotal += report.Total
	}

	reports, total := limitSlice(reports)

	if usageStructuredOutput(tf, reports) {
		return
	}

	printUsageHeader(tf)

	headers := []string{"Day", "App", "Web Transactions", "% of Day", "% of Web", "Top Endpoint"}
	var rows [][]string
	for i, report := range reports {
		if i > 0 {
			rows = append(rows, []string{"", "", "", "", "", ""})
		}
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
				report.Date,
				a.AppName,
				formatTransactions(float64(a.Transactions)),
				fmt.Sprintf("%.1f%%", pctDay),
				fmt.Sprintf("%.1f%%", pctTotal),
				a.TopEndpoint,
			})
		}
	}
	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(len(reports), total)

	if grandTotal > 0 {
		fmt.Printf("Total: %s web transactions\n", formatTransactions(float64(grandTotal)))
	}
	printServerTotal(tf)
}

func runUsageByDaySingleApp(client *api.Client, id int, tf usageTimeframe) {
	from, to := tf.from, tf.to
	chunks := splitTimeframe(from, to)
	showProgress := !jsonOutput

	days := output.RunWithProgress("Fetching usage data", showProgress, func(update func(float64)) []dailyUsage {
		points := fetchAppPoints(client, id, chunks)
		days := bucketByDay(points)

		if len(days) == 0 {
			return days
		}

		// Fetch top endpoint for each day in parallel
		update(0.5)
		var wg sync.WaitGroup
		var done int64
		total := int64(len(days))
		for i := range days {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				d := days[idx]
				dayStart := d.Date + "T00:00:00Z"
				dayEnd := d.Date + "T23:59:59Z"
				endpoints, err := client.ListEndpoints(id, dayStart, dayEnd)
				if err != nil || len(endpoints) == 0 {
					n := atomic.AddInt64(&done, 1)
					update(0.5 + float64(n)/float64(total)*0.5)
					return
				}
				days[idx].TopEndpoint = endpoints[0].Name
				n := atomic.AddInt64(&done, 1)
				update(0.5 + float64(n)/float64(total)*0.5)
			}(i)
		}
		wg.Wait()
		return days
	})

	days, total := limitSlice(days)

	if usageStructuredOutput(tf, days) {
		return
	}

	printUsageHeader(tf)

	headers := []string{"Day", "Web Transactions", "Top Endpoint"}
	rows := make([][]string, len(days))
	var grandTotal int64
	for i := range days {
		d := days[i]
		grandTotal += d.Transactions
		rows[i] = []string{
			d.Date,
			formatTransactions(float64(d.Transactions)),
			d.TopEndpoint,
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(len(days), total)
	printTotalFooter(float64(grandTotal))
	printServerTotal(tf)
}

// usageTimeframe is the resolved window for a usage query. billing is set
// only when --billing-period was used, and carries the org usage payload so
// the billing period can be shown alongside the client-side calculation
// without fetching /usage twice.
type usageTimeframe struct {
	from, to string
	billing  *api.OrgUsage
}

// resolveUsageTimeframe returns the timeframe to use for usage queries.
// With --billing-period it fetches the current billing period from the org
// usage endpoint; otherwise it falls back to the standard --from/--to flags.
func resolveUsageTimeframe(client *api.Client) (usageTimeframe, error) {
	if !billingPeriod {
		from, to, err := resolveTimeframe()
		return usageTimeframe{from: from, to: to}, err
	}

	if fromFlag != "" || toFlag != "" {
		return usageTimeframe{}, fmt.Errorf("--billing-period cannot be used with --from or --to")
	}

	usage, err := client.GetOrgUsage()
	if err != nil {
		return usageTimeframe{}, fmt.Errorf("failed to fetch billing period: %w", err)
	}
	if usage.BillingPeriod.Start == "" || usage.BillingPeriod.End == "" {
		return usageTimeframe{}, fmt.Errorf("billing period not available for this organization")
	}

	start, err := parseBillingTime(usage.BillingPeriod.Start)
	if err != nil {
		return usageTimeframe{}, fmt.Errorf("invalid billing period start: %w", err)
	}
	end, err := parseBillingTime(usage.BillingPeriod.End)
	if err != nil {
		return usageTimeframe{}, fmt.Errorf("invalid billing period end: %w", err)
	}
	// The current period usually ends in the future; there is no data past
	// now, so clamp the query window while still reporting the full period.
	if now := time.Now().UTC(); end.After(now) {
		end = now
	}

	return usageTimeframe{
		from:    start.UTC().Format(time.RFC3339),
		to:      end.UTC().Format(time.RFC3339),
		billing: usage,
	}, nil
}

// billingUsage wraps a usage result with the period it covers and the
// server-reported figure for that period, so --billing-period --json output
// says which window it describes and what the billed total was.
type billingUsage[T any] struct {
	BillingPeriod api.BillingPeriod `json:"billing_period"`
	ServerTotal   *serverTotal      `json:"server_total,omitempty"`
	Usage         T                 `json:"usage"`
}

// serverTotal is the billed transaction figure for the period, which includes
// background jobs and so differs from the per-app web-transaction totals.
type serverTotal struct {
	Transactions int64  `json:"transactions"`
	Limit        *int64 `json:"limit,omitempty"`
}

// usageStructuredOutput emits a usage result as JSON, wrapped with the billing
// period when --billing-period was used. Without it the shape is the bare
// result, unchanged.
func usageStructuredOutput[T any](tf usageTimeframe, data T) bool {
	if tf.billing == nil {
		return structuredOutput(data)
	}
	wrapped := billingUsage[T]{
		BillingPeriod: tf.billing.BillingPeriod,
		Usage:         data,
	}
	if apm := tf.billing.APM; apm != nil {
		wrapped.ServerTotal = &serverTotal{Transactions: apm.TotalTransactions, Limit: apm.Limit}
	}
	return structuredOutput(wrapped)
}

// printUsageHeader prints the billing period when the query is scoped to it,
// otherwise the plain timeframe header.
func printUsageHeader(tf usageTimeframe) {
	if tf.billing == nil {
		printTimeframe(tf.from, tf.to)
		return
	}
	fmt.Printf("%s\n\n", output.DimStyle.Render(fmt.Sprintf("Billing period: %s → %s",
		formatBillingDate(tf.billing.BillingPeriod.Start),
		formatBillingDate(tf.billing.BillingPeriod.End))))
}

// printServerTotal prints the server-reported transaction total for the
// billing period so the client-side calculation can be compared against the
// exact billed figure. No-op unless --billing-period was used.
//
// The per-app figures come from the throughput metric, which counts web
// requests only, whereas the billed total also includes background jobs, so
// the two are expected to differ for apps that run jobs.
func printServerTotal(tf usageTimeframe) {
	if tf.billing == nil || tf.billing.APM == nil {
		return
	}
	fmt.Println(serverTotalLine(tf.billing.APM))
	fmt.Println(output.DimStyle.Render("Per-app totals above count web transactions only; the billed total also includes background jobs."))
}

// serverTotalLine formats the billed transaction total, with the plan limit
// when the API reports one.
func serverTotalLine(apm *api.APMUsage) string {
	line := fmt.Sprintf("Billing period total (server, web + jobs): %s transactions", formatTransactions(float64(apm.TotalTransactions)))
	if apm.Limit != nil && *apm.Limit > 0 {
		line += fmt.Sprintf(" (limit: %s)", formatTransactions(float64(*apm.Limit)))
	}
	return line
}

// fetchAllApps runs fn for each app in parallel and collects the results.
// An optional onProgress callback is called after each app completes with
// (completed, total) counts.
func fetchAllApps[T any](apps []api.App, fn func(api.App) T, onProgress ...func(done, total int)) []T {
	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		results   []T
		completed int64
	)
	total := len(apps)
	for _, app := range apps {
		wg.Add(1)
		go func(a api.App) {
			defer wg.Done()
			result := fn(a)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
			if len(onProgress) > 0 && onProgress[0] != nil {
				n := int(atomic.AddInt64(&completed, 1))
				onProgress[0](n, total)
			}
		}(app)
	}
	wg.Wait()
	return results
}

// fetchAllAppPoints fetches and merges throughput points for all apps in parallel.
func fetchAllAppPoints(client *api.Client, apps []api.App, chunks [][2]string, onProgress ...func(done, total int)) []api.MetricPoint {
	perApp := fetchAllApps(apps, func(a api.App) []api.MetricPoint {
		return fetchAppPoints(client, a.ID, chunks)
	}, onProgress...)
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
		fmt.Printf("\nTotal: %s web transactions\n", formatTransactions(grandTotal))
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
